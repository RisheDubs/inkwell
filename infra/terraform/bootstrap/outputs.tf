output "state_bucket" {
  description = "Bucket name for the backend blocks in every other root."
  value       = aws_s3_bucket.state.bucket
}