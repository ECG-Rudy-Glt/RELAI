### Next
- Fix the contact form (`site/contact.php`): broken since the SMTP password rotation on 2026-06-18. Read `SMTP_PASSWORD` from the environment, log send failures.

### Decisions
- 2026-06-20 — Blog dropped — the owner posts specials on Instagram. `TODO.md` still lists it.
- 2026-06-20 — Dark mode and gallery stay low priority.

### Pitfalls
- Deploy is `make deploy` (rsync); there is no staging.
