resource "aws_cognito_user_pool" "this" {
  name                = var.name
  deletion_protection = var.deletion_protection
  user_pool_tier      = "ESSENTIALS" # same as dev; free up to 10,000 monthly active users
  mfa_configuration   = "OFF"        # the web app has no MFA screens yet

  # Sign in with email; emails are case-insensitive.
  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  username_configuration {
    case_sensitive = false
  }

  # A changed email must be verified before it replaces the old one.
  user_attribute_update_settings {
    attributes_require_verification_before_update = ["email"]
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  admin_create_user_config {
    allow_admin_create_user_only = false # people sign up themselves
  }

  # Cognito's own sender: free, limited to 50 emails/day. Fine for a portfolio app.
  email_configuration {
    email_sending_account = "COGNITO_DEFAULT"
  }

  password_policy {
    minimum_length                   = 8
    require_lowercase                = true
    require_numbers                  = true
    require_symbols                  = true
    require_uppercase                = true
    temporary_password_validity_days = 7
  }

  sign_in_policy {
    allowed_first_auth_factors = ["PASSWORD"]
  }

  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
  }

  # Same email attribute as dev. Schema blocks can't change after creation.
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
}

# Public client for the web app: no secret (a browser can't keep one).
resource "aws_cognito_user_pool_client" "web" {
  name            = "inkwell-web"
  user_pool_id    = aws_cognito_user_pool.this.id
  generate_secret = false

  # SRP only: the password itself is never sent to Cognito. No USER_PASSWORD_AUTH, no USER_AUTH.
  explicit_auth_flows = [
    "ALLOW_USER_SRP_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
  ]

  # No hosted UI / OAuth: Amplify signs in with SRP directly.
  allowed_oauth_flows_user_pool_client = false

  access_token_validity  = 60
  id_token_validity      = 60
  refresh_token_validity = var.refresh_token_days
  auth_session_validity  = 3

  token_validity_units {
    access_token  = "minutes"
    id_token      = "minutes"
    refresh_token = "days"
  }

  enable_token_revocation       = true
  prevent_user_existence_errors = "ENABLED" # sign-in errors don't reveal whether an email exists
}