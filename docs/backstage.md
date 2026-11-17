# Relai and Backstage

**Different problems.** Backstage is a developer portal: a server with a catalogue of your services, owners and
docs, for people. Relai gives an AI agent (and you) the current state of a repository and the next step, from files
in git, with no server. A team that already runs Backstage can keep its catalogue in step with `relai.yaml`.

## Export

```sh
relai export backstage [--type service] [--lifecycle experimental] [--namespace default] [-o catalog-info.yaml] [dir]
```

One-way: Relai never reads Backstage and does not depend on it. It prints a multi-document YAML:

| `relai.yaml` | Backstage |
|---|---|
| `id` | Component `metadata.name` (characters Backstage rejects, such as `/`, become `-`; the original id goes in the annotation `relai.dev/id`) |
| `summary` | `metadata.description` |
| first of `owners` | `spec.owner` (a bare name, which Backstage reads as a group) |
| `provides` | one `API` entity per capability (type `relai-capability`), listed in `spec.providesApis` |
| `consumes` | `spec.consumesApis`, pointing at the API entity of the same capability name |
| `links` | `spec.dependsOn` (components) |
| not in `relai.yaml` | `--type`, `--lifecycle`, `--namespace` (defaults `service`, `experimental`, `default`) |

Commit the result as `catalog-info.yaml` and register it in Backstage as usual.

## What was tested

The output is checked with Backstage's own validators (`@backstage/catalog-model`): see `eval/backstage/`. That proves
the entities are well formed. It does **not** prove how they look in a running Backstage (relations, search, the
`relai-capability` API type, which has no renderer there); that needs a real instance and a platform team's feedback.
Not built: reading Backstage, exporting the whole index in one call, owners as `user:` references.
