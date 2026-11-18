variable "name" {
  description = "Prefix, e.g. inkwell-prod. Bucket becomes <name>-content-<account>."
  type        = string
}

variable "key_description" {
  description = "Description on the KMS key."
  type        = string
}

variable "versioning" {
  description = "Keep previous versions of entry bodies and images."
  type        = bool
  default     = true
}

variable "noncurrent_days" {
  description = "Days to keep previous object versions (only used with versioning)."
  type        = number
  default     = 30
}

variable "key_deletion_window_days" {
  description = "Waiting period before a scheduled key deletion (7-30)."
  type        = number
  default     = 7
}