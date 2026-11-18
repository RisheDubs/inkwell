variable "name" {
  description = "User pool name, e.g. inkwell-prod."
  type        = string
}

variable "deletion_protection" {
  description = "ACTIVE blocks deleting the pool (and its users). Set INACTIVE only for a deliberate destroy."
  type        = string
  default     = "ACTIVE"

  validation {
    condition     = contains(["ACTIVE", "INACTIVE"], var.deletion_protection)
    error_message = "deletion_protection must be ACTIVE or INACTIVE."
  }
}

variable "refresh_token_days" {
  description = "How long a signed-in browser stays signed in."
  type        = number
  default     = 5
}