provider "aws" {
  region = "ap-southeast-2"

  default_tags {
    tags = {
      project    = "inkwell"
      env        = "prod"
      stack      = "foundation"
      managed-by = "terraform"
    }
  }
}

# Credentials come from MONGODB_ATLAS_CLIENT_ID / MONGODB_ATLAS_CLIENT_SECRET
# (source ~/.inkwell-atlas.env). Nothing secret is written here.
provider "mongodbatlas" {}