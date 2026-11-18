output "bucket_name" {
  description = "Content bucket name (S3_BUCKET for the API)."
  value       = aws_s3_bucket.content.bucket
}

output "bucket_arn" {
  description = "Content bucket ARN, for IAM policies."
  value       = aws_s3_bucket.content.arn
}

output "kms_key_arn" {
  description = "Content key ARN, for IAM policies."
  value       = aws_kms_key.content.arn
}