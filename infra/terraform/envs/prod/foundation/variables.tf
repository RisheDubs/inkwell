variable "atlas_org_id" {
  description = "Atlas organization ID (TF_VAR_atlas_org_id from ~/.inkwell-atlas.env)."
  type        = string
}

variable "budget_email" {
  description = "Where budget alerts go (TF_VAR_budget_email from ~/.inkwell-atlas.env)."
  type        = string
}