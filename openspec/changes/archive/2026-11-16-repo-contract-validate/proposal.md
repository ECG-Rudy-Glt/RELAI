## Why

`relai.yaml` is the file every other feature reads (index, find, workspace, links), yet it had no
specification: only a draft in DESIGN §6 and the behaviour of one parser. Nobody else could write a
reader, and a mistyped key (`provide:`) was ignored without a word.

## What Changes

- `spec/relai-yaml.md`: the format on one page, with `apiVersion` and a compatibility rule;
  `spec/relai.schema.json`; conformance cases in `spec/conformance/`; a second reader in Python.
- `relai validate [--strict] [--json] [dir]`: checks `relai.yaml`, the size of `AGENTS.md` and the
  presence of `.relai/notes.md`. Exit code 1 on an error, made for CI.
- The reader keeps `conventions:` as text; the README documents a session-start snippet for agents
  other than Claude Code.

## Impact

New package `internal/validate`, new command, new files under `spec/`. No change to existing output.
