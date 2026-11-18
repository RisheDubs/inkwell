# Customer-managed key for the dev content bucket. Rotation is on (yearly).
resource "aws_kms_key" "content" {
  description              = "Inkwell dev: entry bodies and images"
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  key_usage                = "ENCRYPT_DECRYPT"
  enable_key_rotation      = true
  rotation_period_in_days  = 365
  multi_region             = false

  # Default key policy: the account's IAM policies decide who can use the key.
  policy = jsonencode({
    Version = "2012-10-17"
    Id      = "key-default-1"
    Statement = [{
      Sid       = "Enable IAM User Permissions"
      Effect    = "Allow"
      Principal = { AWS = "arn:aws:iam::658586621994:root" }
      Action    = "kms:*"
      Resource  = "*"
    }]
  })

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_kms_alias" "content" {
  name          = "alias/inkwell-dev-content"
  target_key_id = aws_kms_key.content.key_id
}

resource "aws_s3_bucket" "content" {
  bucket              = "inkwell-dev-content-658586621994"
  bucket_namespace    = "global" # as imported; kept so the import can't plan a replacement
  force_destroy       = false
  object_lock_enabled = false

  lifecycle {
    prevent_destroy = true
  }
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

# Dev never had versioning on; prod will (ADR-0011 keeps the import a no-op).
resource "aws_s3_bucket_versioning" "content" {
  bucket = aws_s3_bucket.content.id

  versioning_configuration {
    status     = "Disabled"
    mfa_delete = "Disabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "content" {
  bucket = aws_s3_bucket.content.id

  rule {
    bucket_key_enabled       = true
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
}

# Browser image uploads (presigned POST) from the local web app. Replaces infra/dev/s3-cors.json.
resource "aws_s3_bucket_cors_configuration" "content" {
  bucket = aws_s3_bucket.content.id

  cors_rule {
    id              = "inkwell-web-dev"
    allowed_methods = ["POST"]
    allowed_origins = ["http://localhost:5173"]
    allowed_headers = []
    expose_headers  = []
    max_age_seconds = 3000
  }
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