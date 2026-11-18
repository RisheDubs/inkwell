# Dev content bucket and key, built from the same module as prod (ADR-0011).
module "content" {
  source = "../../modules/storage"

  name            = "inkwell-dev"
  key_description = "Inkwell dev: entry bodies and images"
  versioning      = false # dev never had versioning; prod does
}

# Browser image uploads (presigned POST) from the local web app.
resource "aws_s3_bucket_cors_configuration" "content" {
  bucket = module.content.bucket_name

  cors_rule {
    id              = "inkwell-web-dev"
    allowed_methods = ["POST"]
    allowed_origins = ["http://localhost:5173"]
    allowed_headers = []
    expose_headers  = []
    max_age_seconds = 3000
  }
}
