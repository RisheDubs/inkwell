data "aws_caller_identity" "current" {}

# Customer-managed key for entry bodies and images. No policy here: AWS's default key policy
# lets this account's IAM policies decide access, and the API role is granted use through S3 only.
resource "aws_kms_key" "content" {
  description              = var.key_description
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  key_usage                = "ENCRYPT_DECRYPT"
  enable_key_rotation      = true
  rotation_period_in_days  = 365
  multi_region             = false
  deletion_window_in_days  = var.key_deletion_window_days
}

resource "aws_kms_alias" "content" {
  name          = "alias/${var.name}-content"
  target_key_id = aws_kms_key.content.key_id
}

resource "aws_s3_bucket" "content" {
  bucket        = "${var.name}-content-${data.aws_caller_identity.current.account_id}"
  force_destroy = false # a bucket holding user data must be emptied on purpose first
}

resource "aws_s3_bucket_ownership_controls" "content" {
  bucket = aws_s3_bucket.content.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "content" {
  bucket = aws_s3_bucket.content.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "content" {
  bucket = aws_s3_bucket.content.id

  versioning_configuration {
    status     = var.versioning ? "Enabled" : "Disabled"
    mfa_delete = "Disabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "content" {
  bucket = aws_s3_bucket.content.id

  rule {
    bucket_key_enabled       = true # one data key per bucket, far fewer (billed) KMS calls
    blocked_encryption_types = ["SSE-C"]

    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.content.arn
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "content" {
  bucket = aws_s3_bucket.content.id

  rule {
    id     = "abort-incomplete-uploads"
    status = "Enabled"

    abort_incomplete_multipart_upload {
      days_after_initiation = 1
    }
  }

  dynamic "rule" {
    for_each = var.versioning ? [1] : []

    content {
      id     = "expire-old-versions"
      status = "Enabled"

      filter {
        prefix = ""
      }

      noncurrent_version_expiration {
        noncurrent_days = var.noncurrent_days
      }

      expiration {
        expired_object_delete_marker = true
      }
    }
  }

  depends_on = [aws_s3_bucket_versioning.content]
}

resource "aws_s3_bucket_policy" "content" {
  bucket = aws_s3_bucket.content.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "DenyInsecureTransport"
      Effect    = "Deny"
      Principal = "*"
      Action    = "s3:*"
      Resource  = [aws_s3_bucket.content.arn, "${aws_s3_bucket.content.arn}/*"]
      Condition = { Bool = { "aws:SecureTransport" = "false" } }
    }]
  })

  depends_on = [aws_s3_bucket_public_access_block.content]
}