# `relai.yaml` — format v0

One file at the root of a repository that says what the repository is and what it offers to the
others. This page is the whole specification: a reader can be written from it alone (two exist,
[`internal/index`](../internal/index/index.go) in Go and [`readers/relai_yaml.py`](readers/relai_yaml.py)
in Python, both checked against [`conformance/`](conformance/)). The data model is also given as a
JSON Schema: [`relai.schema.json`](relai.schema.json).

## 1. Example

```yaml
apiVersion: relai.dev/v0
id: invoice-pdf
summary: "Generates invoice PDFs from JSON. Used by billing and sales tools."
owners: [alice, team-billing]
provides:
  - capability: pdf.generate.invoice
consumes:
  - capability: auth.oidc
links: [billing-api]
```

## 2. Keys

| Key | Required | Value | Meaning |
|---|---|---|---|
| `apiVersion` | recommended | `relai.dev/v0` | The format version. A missing key means `relai.dev/v0`. |
| `id` | **yes** | a name: letters, digits, `.`, `_`, `-`, `/`; starts with a letter or digit | The repository's name, unique in an index. |
| `summary` | **yes** | one line of text | One sentence that lets a person or an agent decide whether to reuse this repository. |
| `owners` | no | list of names | Who to ask. Free text: a handle, a team. |
| `provides` | no | list of capabilities | What this repository offers to others. |
| `consumes` | no | list of capabilities | What it needs from others. |
| `links` | no | list of `id`s | The repositories it works with, declared by a person. |
| `conventions` | no | text | Reserved: the shared conventions the repository follows. Readers keep it as text. |
| `tasks` | no | `beads` or `github` | Where the tasks live, when the files do not show it. `github` lets `relai handoff` list the open GitHub issues with the `gh` command, the only network call it can make. |

A **capability** is a dotted lowercase name going from general to specific, such as `auth.oidc` or
`pdf.generate.invoice`: lowercase letters, digits, `.`, `_`, `-`; starts with a letter or digit.

## 3. Syntax

`relai.yaml` is valid YAML, restricted to a subset small enough to read without a YAML library:

- Top-level keys start in the first column: `key: value`. A value may be quoted with `"` or `'`.
- `#` outside quotes starts a comment. Blank lines are ignored.
- A list of names is written inline, `[a, b]`, or as a block, one `- a` per line.
- An entry of `provides` or `consumes` is either a bare name or a mapping with a `capability:` key,
  on one line (`- capability: a`, `{capability: a}`) or on several, the key in any position:

  ```yaml
  provides:
    - id: invoices            # other keys of an entry are ignored
      capability: pdf.generate.invoice
  ```
- `[]`, `~`, `null` or nothing after the colon is an empty list.

Anything else (anchors, multi-line strings, nested mappings beyond the above) is outside the subset.

## 4. Compatibility rule

1. **Readers ignore what they do not know.** An unknown top-level key, or an unknown key inside a
   `provides`/`consumes` entry, is skipped without an error. That is how the format grows.
2. **Readers never drop silently what they should know.** A value of a known key that cannot be
   understood is reported (a warning naming the key), never treated as absent.
3. **Within `v0`, changes are additive.** A new optional key does not change `apiVersion`.
4. **The version changes** (`relai.dev/v1`) when a required key is added, a key is removed or
   renamed, or the meaning of an existing key changes.
5. **A reader that meets a version it does not know** says so and does not index the file. It does
   not guess.

## 5. The `.relai/` folder

| Path | Written by | In git | Content |
|---|---|---|---|
| `.relai/notes.md` | people and agents | yes | Decisions, pitfalls, what to do next. Read by `relai handoff`. |
| `.relai/notes-archive.md` | `relai note archive` | yes | Entries moved out of `notes.md`. Not read by the handoff. |
| `.relai/index.json` | `relai index build` | your choice | The collected `relai.yaml` of a group of repositories. Read by `relai find` and `relai workspace`. |
| `.relai/handoff.md` | `relai handoff -o` | no (generated) | A saved copy of the handoff, for tools that read a file. |

Nothing else is reserved. A tool may add its own files under `.relai/`; readers ignore them.

## 6. Checking a file

`relai validate` checks a repository against this page: required keys, names, values that were not
understood, unknown keys (reported, since a typo such as `provide:` would otherwise be ignored
without a word), and the size of `AGENTS.md`.
