resource "mongodbatlas_project" "this" {
  name   = var.project_name
  org_id = var.org_id
}

# M0: free, shared, on AWS in Sydney. One M0 per project.
resource "mongodbatlas_advanced_cluster" "this" {
  project_id   = mongodbatlas_project.this.id
  name         = var.cluster_name
  cluster_type = "REPLICASET"

  replication_specs = [{
    region_configs = [{
      provider_name         = "TENANT"
      backing_provider_name = "AWS"
      region_name           = var.region
      priority              = 7
      electable_specs = {
        instance_size = "M0"
      }
    }]
  }]
}

# 32 letters and digits (~190 bits). No symbols, so it needs no escaping inside a URI.
resource "random_password" "db" {
  length  = 32
  special = false
}

resource "mongodbatlas_database_user" "api" {
  project_id         = mongodbatlas_project.this.id
  username           = var.db_username
  password           = random_password.db.result
  auth_database_name = "admin"

  # Least privilege: one database, nothing else.
  roles {
    role_name     = "readWrite"
    database_name = var.database_name
  }

  # Only valid on this cluster.
  scopes {
    name = mongodbatlas_advanced_cluster.this.name
    type = "CLUSTER"
  }
}

resource "mongodbatlas_project_ip_access_list" "this" {
  for_each = toset(var.access_cidrs)

  project_id = mongodbatlas_project.this.id
  cidr_block = each.value
  comment    = "Inkwell API (Fargate public IPs change per deploy; see ADR-0012)"
}

locals {
  # mongodb+srv://cluster.xxxx.mongodb.net -> mongodb+srv://user:pass@cluster.xxxx.mongodb.net/
  srv_host = trimprefix(mongodbatlas_advanced_cluster.this.connection_strings.standard_srv, "mongodb+srv://")
  uri = format(
    "mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority&appName=inkwell-api",
    var.db_username,
    random_password.db.result,
    local.srv_host,
  )
}