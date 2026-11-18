provider "aws" {
  region = "ap-southeast-2"

  # Imported resources pick these up as an in-place tag update on the first apply.
  default_tags {
    tags = {
      project    = "inkwell"
      env        = "dev"
      stack      = "dev"
      managed-by = "terraform"
    }
  }
}