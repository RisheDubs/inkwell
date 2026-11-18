module "network" {
  source = "../../../modules/network"

  name = "inkwell-prod"
}

module "ecr" {
  source = "../../../modules/ecr"

  name = "inkwell-prod-api"
}

module "storage" {
  source = "../../../modules/storage"

  name            = "inkwell-prod"
  key_description = "Inkwell prod: entry bodies and images"
}

module "cognito" {
  source = "../../../modules/cognito"

  name = "inkwell-prod"
}

module "atlas" {
  source = "../../../modules/atlas"

  org_id       = var.atlas_org_id
  project_name = "inkwell-prod"
  cluster_name = "inkwell-prod"
}