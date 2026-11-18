variable "name" {
  description = "Repository name, e.g. inkwell-prod-api."
  type        = string
}

variable "keep_images" {
  description = "How many tagged images to keep."
  type        = number
  default     = 10
}