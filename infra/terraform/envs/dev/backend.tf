terraform {
  backend "s3" {
    bucket       = "inkwell-tfstate-658586621994"
    key          = "dev/terraform.tfstate"
    region       = "ap-southeast-2"
    encrypt      = true
    use_lockfile = true
  }
}