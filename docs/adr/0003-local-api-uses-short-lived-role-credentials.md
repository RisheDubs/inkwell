# ADR-0003: The local API uses short-lived role credentials

- Status: Accepted
- Date: 2026-11-10
- Deciders: Rishekesh

## Context

From Phase 3 the API talks to real AWS: it writes entry bodies to S3 and needs a KMS key to encrypt them.
On my laptop the API runs in Docker, so it needs AWS credentials from somewhere. The two easy options
both worried me. My own login (`rishe`) is in the Admins group, so a container holding it could do
anything in my account. An access key never expires, which is exactly the kind of secret that ends up
in git history.

## Decision

I created an IAM role, `inkwell-dev-api`, with only what the API needs: get, put and delete objects
under `users/` in the dev content bucket, and use the content KMS key only through S3. My `rishe`
session assumes that role (`make creds`, which wraps `aws configure export-credentials`), and the
container receives credentials that expire after one hour through a gitignored `.aws-session.env` file.

## Consequences

- The container can do exactly what my production API's task role will be able to do, so permission
  bugs show up on my laptop instead of after a deploy. In Phase 5 the ECS task role gets the same
  policy and the Go code doesn't change: the AWS SDK finds credentials either way.
- No long-lived AWS keys exist anywhere for this project.
- Every hour I run `make creds && make up` again. It's a small annoyance I'm accepting on purpose.
- The session file holds live credentials for up to an hour, so it is gitignored and only readable by me.

## Alternatives considered

- **Mount `~/.aws` into the container.** That hands the container my admin session. Rejected.
- **An access key for a dedicated IAM user.** Least privilege, but the key never expires and I'd have
  to rotate it by hand. Rejected.
- **LocalStack instead of real S3.** Free and offline, but it doesn't enforce IAM and KMS the way AWS
  does, and testing those is the point of this phase. The build plan already chose real AWS for dev.