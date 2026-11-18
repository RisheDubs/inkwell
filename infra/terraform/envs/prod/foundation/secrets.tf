# MONGODB_URI for the API task. ECS reads it at start-up by ARN (step 12); the value never
# appears in the task definition, the image or the logs.
resource "aws_secretsmanager_secret" "mongodb_uri" {
  name        = "inkwell/prod/mongodb-uri"
  description = "Atlas connection string for the Inkwell API (ADR-0012)"

  # The AWS-managed key (aws/secretsmanager) is free; a customer-managed key would add $1/month.
  # Delete immediately on destroy so the name can be reused; Terraform can always rebuild the value.
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "mongodb_uri" {
  secret_id     = aws_secretsmanager_secret.mongodb_uri.id
  secret_string = module.atlas.mongodb_uri
}