variable "org_id" {
  description = "Atlas organization ID."
  type        = string
}

variable "project_name" {
  description = "Atlas project name."
  type        = string
}

variable "cluster_name" {
  description = "Cluster name (letters, numbers, hyphens)."
  type        = string
}

variable "region" {
  description = "Atlas region name for AWS."
  type        = string
  default     = "AP_SOUTHEAST_2"
}

variable "database_name" {
  description = "The only database the API user can read and write."
  type        = string
  default     = "inkwell"
}

variable "db_username" {
  description = "Database user for the API."
  type        = string
  default     = "inkwell-api"
}

variable "access_cidrs" {
  description = "CIDRs allowed to connect. M0 has no private networking (ADR-0012)."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}