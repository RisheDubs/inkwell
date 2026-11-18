output "vpc_id" {
  description = "VPC ID."
  value       = aws_vpc.this.id
}

output "public_subnet_ids" {
  description = "Subnets for Fargate tasks (public IP, egress only)."
  value       = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  description = "Subnets for the internal ALB."
  value       = aws_subnet.private[*].id
}

output "alb_security_group_id" {
  description = "Security group for the internal ALB."
  value       = aws_security_group.alb.id
}

output "task_security_group_id" {
  description = "Security group for API tasks."
  value       = aws_security_group.task.id
}