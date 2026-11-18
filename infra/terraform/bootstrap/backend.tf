# State for this stack lives in the bucket it creates (migrated after the first apply).
# Bucket names aren't secrets; credentials are never written here.
terraform {
  backend "s3" {
    bucket       = "inkwell-tfstate-658586621994"
    key          = "bootstrap/terraform.tfstate"
    region       = "ap-southeast-2"
    encrypt      = true
    use_lockfile = true # S3 native locking: a .tflock object, no DynamoDB
  }
}