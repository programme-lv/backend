# User email (password reset / verification)

Cross-repo behaviour and ops: [`docs/github/transactional-email.md`](../../../../docs/github/transactional-email.md) (sibling docs repo).

This module owns:

- `email_verified` on `users`
- `user_email_tokens` (hashed one-time tokens)
- SES SMTP mailer under `modules/user/mail/`
- HTTP: `/password-reset/*`, `/email-verification/*`

Templates live in `modules/user/mail/templates/`. Iterate locally with Mailpit; do not treat SES console templates as source of truth.
