module "network" {
  source = "../../../modules/network"

  name = "inkwell-prod"
}

module "ecr" {
  source = "../../../modules/ecr"

  name = "inkwell-prod-api"
}

module "storage" {
  source = "../../../modules/storage"

  name            = "inkwell-prod"
  key_description = "Inkwell prod: entry bodies and images"
}