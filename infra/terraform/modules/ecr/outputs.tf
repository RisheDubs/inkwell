output "repository_url" {
  description = "Registry URL to push to and to put in the task definition."
  value       = aws_ecr_repository.this.repository_url
}

output "repository_arn" {
  description = "Repository ARN, for the task execution role's pull permission."
  value       = aws_ecr_repository.this.arn
}