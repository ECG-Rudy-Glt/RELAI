# Prototypes

Throwaway code used to test ideas before the real implementation. Not stable, not packaged.

- `handoff.py` — experiment E1 (see [design §10](../docs/DESIGN.md)): builds a compact handoff
  from git, Beads (`bd --json`), OpenSpec (`openspec list --json`) and `.relai/notes.md`.
  Python 3 standard library only. Usage: `python3 prototype/handoff.py /path/to/repo`.
