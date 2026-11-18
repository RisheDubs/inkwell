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