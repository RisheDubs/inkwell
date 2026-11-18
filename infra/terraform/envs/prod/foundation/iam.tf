data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

locals {
  account_id = data.aws_caller_identity.current.account_id
  region     = data.aws_region.current.region
}

# API logs. Kept in foundation so they survive runtime being destroyed.
resource "aws_cloudwatch_log_group" "api" {
  name              = "/ecs/inkwell-prod-api"
  retention_in_days = 7
}

# Only ECS tasks in this account may assume these roles (confused-deputy protection).
data "aws_iam_policy_document" "ecs_tasks_assume" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = [local.account_id]
    }

    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values   = ["arn:aws:ecs:${local.region}:${local.account_id}:*"]
    }
  }
}

# --- Execution role: used by ECS before the app starts (pull image, write logs, read the secret) ---

resource "aws_iam_role" "execution" {
  name               = "inkwell-prod-api-execution"
  description        = "ECS: pull the API image, write its logs, inject MONGODB_URI"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_assume.json
}

data "aws_iam_policy_document" "execution" {
  # ECR login can't be scoped to a repository.
  statement {
    sid       = "EcrAuthToken"
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }

  statement {
    sid       = "PullApiImageOnly"
    actions   = ["ecr:BatchGetImage", "ecr:GetDownloadUrlForLayer", "ecr:BatchCheckLayerAvailability"]
    resources = [module.ecr.repository_arn]
  }

  statement {
    sid       = "WriteApiLogsOnly"
    actions   = ["logs:CreateLogStream", "logs:PutLogEvents"]
    resources = ["${aws_cloudwatch_log_group.api.arn}:*"]
  }

  # The secret uses the AWS-managed key, so no kms:Decrypt grant is needed.
  statement {
    sid       = "ReadMongoUriOnly"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.mongodb_uri.arn]
  }
}

resource "aws_iam_role_policy" "execution" {
  name   = "inkwell-prod-api-execution"
  role   = aws_iam_role.execution.id
  policy = data.aws_iam_policy_document.execution.json
}

# --- Task role: what the API code itself may do ---

resource "aws_iam_role" "task" {
  name               = "inkwell-prod-api-task"
  description        = "Inkwell API: entry bodies and media under users/ in the prod content bucket"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_assume.json
}

data "aws_iam_policy_document" "task" {
  # The code only calls GetObject and PutObject (presigned POST is signed as PutObject).
  statement {
    sid       = "EntryBodiesAndMediaOnly"
    actions   = ["s3:GetObject", "s3:PutObject"]
    resources = ["${module.storage.bucket_arn}/users/*"]
  }

  statement {
    sid       = "UseContentKeyOnlyThroughS3"
    actions   = ["kms:GenerateDataKey", "kms:Decrypt"]
    resources = [module.storage.kms_key_arn]

    condition {
      test     = "StringEquals"
      variable = "kms:ViaService"
      values   = ["s3.${local.region}.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy" "task" {
  name   = "inkwell-prod-api-content"
  role   = aws_iam_role.task.id
  policy = data.aws_iam_policy_document.task.json
}