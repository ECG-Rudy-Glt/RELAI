#!/bin/sh
# Runs when a Claude Code session starts (see .claude/settings.json). Whatever it prints becomes part of
# the session's context, so the agent starts from the project's current state instead of guessing.
# It never fails: a broken hook must not block a session.
cd "${CLAUDE_PROJECT_DIR:-.}" 2>/dev/null || exit 0
# Hooks can run without the login shell's PATH (an editor extension, a desktop app): add, after it,
# the folders where relai, bd and go install their binaries.
PATH="$PATH:$HOME/.local/bin:$HOME/go/bin:/usr/local/bin"; export PATH
if command -v relai >/dev/null 2>&1; then
  if relai handoff 2>/dev/null; then exit 0; fi
  echo "# Handoff (relai handoff failed here, so this is read from the plain files)"
else
  echo "# Handoff (the relai CLI was not found, so this is read from the plain files)"
fi
for f in .relai/notes.md relai.yaml; do
  if [ -f "$f" ]; then echo; echo "## $f"; cat "$f"; fi
done
exit 0
