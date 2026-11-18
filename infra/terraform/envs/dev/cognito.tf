# Dev user pool, made in the console in Phase 2 and imported (ADR-0011).
# The name is the console default; renaming a pool replaces it, which would delete every user.
resource "aws_cognito_user_pool" "dev" {
  name                     = "User pool - 0yltmo"
  deletion_protection      = "ACTIVE"
  user_pool_tier           = "ESSENTIALS"
  mfa_configuration        = "OFF"
  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
    recovery_mechanism {
      name     = "verified_phone_number"
      priority = 2
    }
  }

  admin_create_user_config {
    allow_admin_create_user_only = false
  }

  email_configuration {
    email_sending_account = "COGNITO_DEFAULT"
  }

  password_policy {
    minimum_length                   = 8
    password_history_size            = 0
    require_lowercase                = true
    require_numbers                  = true
    require_symbols                  = true
    require_uppercase                = true
    temporary_password_validity_days = 7
  }

  schema {
    name                     = "email"
    attribute_data_type      = "String"
    developer_only_attribute = false
    mutable                  = true
    required                 = true

    string_attribute_constraints {
      min_length = "0"
      max_length = "2048"
    }
  }

  sign_in_policy {
    allowed_first_auth_factors = ["PASSWORD"]
  }

  username_configuration {
    case_sensitive = false
  }

  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
  }

  lifecycle {
    prevent_destroy = true
  }
}

# Public client (no secret) used by the web app.
# Dev keeps USER_PASSWORD_AUTH/USER_AUTH for the curl token block; prod will not.
# The OAuth settings are console-wizard leftovers that Amplify (SRP) never uses.
resource "aws_cognito_user_pool_client" "web" {
  name         = "inkwell-web"
  user_pool_id = aws_cognito_user_pool.dev.id

  explicit_auth_flows = [
    "ALLOW_REFRESH_TOKEN_AUTH",
    "ALLOW_USER_AUTH",
    "ALLOW_USER_PASSWORD_AUTH",
    "ALLOW_USER_SRP_AUTH",
  ]

  access_token_validity  = 60
  id_token_validity      = 60
  refresh_token_validity = 5
  auth_session_validity  = 3

  token_validity_units {
    access_token  = "minutes"
    id_token      = "minutes"
    refresh_token = "days"
  }

  enable_token_revocation                       = true
  prevent_user_existence_errors                 = "ENABLED"
  enable_propagate_additional_user_context_data = false

  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code"]
  allowed_oauth_scopes                 = ["email", "openid", "phone"]
  callback_urls                        = ["http://localhost:5173/callback"]
  logout_urls                          = []
  supported_identity_providers         = ["COGNITO"]
  read_attributes                      = []
  write_attributes                     = []
}