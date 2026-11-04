# ADR-0001: Signing in to AWS with my IAM user and `aws login`

- **Status:** Accepted
- **Date:** 2026-11-04

## Context

My build plan said to sign in to AWS through IAM Identity Center rather than the root user. The reasoning behind that is simple: nobody should be using root day to day, and I don't want long-lived access keys sitting on my laptop.

When I got to this step, I realised I already had an admin IAM user (`rishe`) with MFA turned on. I'd also been signed in as root without realising it. The console showed "rishe" in the corner, and it was only when I ran `aws sts get-caller-identity` in CloudShell and saw `:root` in the Arn that I caught it. That was a useful lesson in checking identity properly instead of trusting a display name.

The traditional catch with IAM users is the CLI: you'd normally create access keys,
which are basically passwords that never expire, stored in a plain-text file. That's
exactly what I wanted to avoid. But I found that AWS released `aws login` in late 2025.
It lets me sign in through the browser with my IAM user and MFA, and gives the CLI
temporary credentials that refresh every 15 minutes and expire within 12 hours. No
access keys at all.

So I had three options in front of me:

1. **Set up IAM Identity Center**, as the plan suggested. It's what AWS recommends for people, but it means creating an AWS Organization and more setup for an account
that only I use.
2. **Use my existing IAM user with `aws login`.** It meets the same security goal with what I already had.
3. **Just use root.** I'll admit I was tempted, but I ruled it out. Root can't be
restricted by any policy, and if that session were ever hijacked, someone could run
up a huge bill, delete everything, or lock me out of my own account.

## Decision

I'm using my IAM user `rishe` for the AWS console (MFA required, with permissions
coming from an `Admins` group rather than attached directly to me), and `aws login --profile inkwell` for the CLI, Terraform and the Go SDK. I won't create access keys for myself, and root stays locked away for the rare tasks that genuinely need it.

## Consequences

**What I gained**

- No long-lived credentials on my machine. Everything the CLI uses is temporary and tied to MFA.
- I got to skip setting up an Organization I don't need yet for a one-person sandbox.
- I learned a brand-new AWS feature along the way (it needs AWS CLI 2.32.0+; I'm on 2.39.3).

**What I gave up**

- AWS generally steers people away from IAM users for human access, and I'm not getting hands-on practice with Identity Center and permission sets, which are common in larger companies.
- My user has broad admin rights. I'm comfortable with that because I'm the only person here, it's protected by MFA, and I'm applying strict least privilege where it matters most: the machines.
- If someone else joins, I'd be managing individual IAM users instead of central SSO.

**When I'd revisit this:** if another person joins the project, if I split into
separate dev and prod accounts, or if I need centralised access auditing. Any of those would make Identity Center the better choice, and I'd write a new ADR to replace this one.