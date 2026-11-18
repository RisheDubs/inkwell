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