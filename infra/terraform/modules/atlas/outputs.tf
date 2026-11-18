output "project_id" {
  description = "Atlas project ID."
  value       = mongodbatlas_project.this.id
}

output "srv_host" {
  description = "Cluster host for mongodb+srv (no credentials)."
  value       = local.srv_host
}

output "mongodb_uri" {
  description = "Full connection string with credentials. Goes into Secrets Manager (step 10)."
  value       = local.uri
  sensitive   = true
}