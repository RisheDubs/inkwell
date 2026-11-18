module "network" {
  source = "../../../modules/network"

  name = "inkwell-prod"
}

module "ecr" {
  source = "../../../modules/ecr"

  name = "inkwell-prod-api"
}