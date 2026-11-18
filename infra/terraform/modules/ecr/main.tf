resource "aws_ecr_repository" "this" {
  name = var.name

  # Tags are git SHAs; a tag can never be overwritten with different code.
  image_tag_mutability = "IMMUTABLE"

  # Images are rebuildable from git, so a full destroy may delete them.
  force_delete = true

  # Free basic scanning on every push.
  image_scanning_configuration {
    scan_on_push = true
  }

  # AES256 is free; a KMS key would add $1/month.
  encryption_configuration {
    encryption_type = "AES256"
  }
}

resource "aws_ecr_lifecycle_policy" "this" {
  repository = aws_ecr_repository.this.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Delete untagged images after 1 day"
        selection = {
          tagStatus   = "untagged"
          countType   = "sinceImagePushed"
          countUnit   = "days"
          countNumber = 1
        }
        action = { type = "expire" }
      },
      {
        rulePriority = 2
        description  = "Keep only the newest ${var.keep_images} images"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = var.keep_images
        }
        action = { type = "expire" }
      },
    ]
  })
}