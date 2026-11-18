# Read by envs/prod/runtime through terraform_remote_state.
output "vpc_id" {
  description = "Prod VPC."
  value       = module.network.vpc_id
}

output "public_subnet_ids" {
  description = "Subnets for Fargate tasks."
  value       = module.network.public_subnet_ids
}

output "private_subnet_ids" {
  description = "Subnets for the internal ALB."
  value       = module.network.private_subnet_ids
}

output "alb_security_group_id" {
  description = "Internal ALB security group."
  value       = module.network.alb_security_group_id
}

output "task_security_group_id" {
  description = "API task security group."
  value       = module.network.task_security_group_id
}

output "ecr_repository_url" {
  description = "API image repository."
  value       = module.ecr.repository_url
}

output "ecr_repository_arn" {
  description = "API image repository ARN."
  value       = module.ecr.repository_arn
}

output "content_bucket_name" {
  description = "Prod content bucket."
  value       = module.storage.bucket_name
}

output "content_bucket_arn" {
  description = "Prod content bucket ARN."
  value       = module.storage.bucket_arn
}

output "content_kms_key_arn" {
  description = "Prod content KMS key ARN."
  value       = module.storage.kms_key_arn
}

output "cognito_user_pool_id" {
  description = "Prod user pool."
  value       = module.cognito.user_pool_id
}

output "cognito_client_id" {
  description = "Prod web client (public)."
  value       = module.cognito.client_id
}

output "atlas_srv_host" {
  description = "Atlas cluster host (no credentials)."
  value       = module.atlas.srv_host
}