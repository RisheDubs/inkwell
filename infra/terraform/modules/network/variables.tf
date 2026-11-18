variable "name" {
  description = "Prefix for resource names, e.g. inkwell-prod."
  type        = string
}

variable "cidr" {
  description = "VPC CIDR block."
  type        = string
  default     = "10.20.0.0/16"
}

variable "az_count" {
  description = "Number of availability zones (an ALB needs at least 2)."
  type        = number
  default     = 2

  validation {
    condition     = var.az_count >= 2
    error_message = "An ALB needs subnets in at least 2 availability zones."
  }
}

variable "app_port" {
  description = "Port the API container listens on."
  type        = number
  default     = 8080
}