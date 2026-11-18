output "user_pool_id" {
  description = "COGNITO_USER_POOL_ID for the API and the web build."
  value       = aws_cognito_user_pool.this.id
}

output "client_id" {
  description = "COGNITO_CLIENT_ID for the API and the web build (public, not a secret)."
  value       = aws_cognito_user_pool_client.web.id
}