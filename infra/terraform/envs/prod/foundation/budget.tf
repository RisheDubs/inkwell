# Catches a forgotten runtime (about US$1.10/day) within a day, long before the monthly budget.
# AWS refreshes cost data a few times a day, so the alert can lag by some hours.
resource "aws_budgets_budget" "daily" {
  name         = "inkwell-daily"
  budget_type  = "COST"
  limit_amount = "2"
  limit_unit   = "USD"
  time_unit    = "DAILY"

  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 100
    threshold_type             = "PERCENTAGE"
    notification_type          = "ACTUAL"
    subscriber_email_addresses = [var.budget_email]
  }
}