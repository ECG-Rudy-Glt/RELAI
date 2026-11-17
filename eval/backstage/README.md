# Backstage export check

Validates what `relai export backstage` prints with Backstage's own code (`@backstage/catalog-model`):
the entity envelope, the entity name and namespace rules, and the Component and API kind validators.
It does not start a Backstage instance (see `docs/backstage.md`).

```sh
mkdir /tmp/bs && cd /tmp/bs && npm init -y && npm install @backstage/catalog-model yaml
relai export backstage /path/to/repo > catalog-info.yaml
node /path/to/RELAI/eval/backstage/check.mjs catalog-info.yaml     # "N entities checked, 0 problems"
```

Last run, 2026-11-17, `@backstage/catalog-model` from npm: the spec's `invoice-pdf` example (with a `/` in its id,
owners, provides, consumes and links) and Relai's own `relai.yaml`: 5 entities, 0 problems. A name with a
`/` and a space is rejected, so the check does fail on bad input.
