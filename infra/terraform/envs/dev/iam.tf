# Role the local API assumes (make creds). Only Rishe can assume it, for 1 hour at most.
resource "aws_iam_role" "api" {
  name                 = "inkwell-dev-api"
  description          = "Inkwell API (local dev): entry bodies and media in the dev content bucket"
  path                 = "/"
  max_session_duration = 3600

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "RisheAssumesForLocalDev"
      Effect    = "Allow"
      Principal = { AWS = "arn:aws:iam::658586621994:user/Rishe" }
      Action    = "sts:AssumeRole"
    }]
  })
}

# Least privilege: objects under users/ only, and the key only when S3 calls it.
resource "aws_iam_role_policy" "api_content" {
  name = "inkwell-dev-api-content"
  role = aws_iam_role.api.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "EntryBodiesAndMediaOnly"
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
        Resource = "${aws_s3_bucket.content.arn}/users/*"
      },
      {
        Sid       = "UseContentKeyOnlyThroughS3"
        Effect    = "Allow"
        Action    = ["kms:GenerateDataKey", "kms:Decrypt"]
        Resource  = aws_kms_key.content.arn
        Condition = { StringEquals = { "kms:ViaService" = "s3.ap-southeast-2.amazonaws.com" } }
      },
    ]
  })
}