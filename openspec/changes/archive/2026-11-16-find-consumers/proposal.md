# Proposal

## Why

`relai find <capability>` answers "who provides it", which is the question before building something.
The question before changing or removing a capability is the opposite: "who depends on it". Each
`relai.yaml` already declares `consumes:`, and the index already stores it, but nothing reads it back.

## What Changes

- New flag `relai find --consumers [--index <path>] <capability>`: lists the indexed repositories that
  declare the capability under `consumes:`, with the same matching rule and output style as the
  providers lookup.
- New flag `relai find --json` (both modes): `{"capability", "mode", "matches"}`, `[]` when empty.
- Finding nothing is stated plainly and exits 0, like the providers lookup.

## Impact

- Existing `relai find` output and exit codes are unchanged.
- The answer is only as complete as what repositories declare; the output says so when empty.
