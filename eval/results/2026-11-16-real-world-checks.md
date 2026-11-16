# Real-world checks (2026-11-16)

Task relai-31i: things never checked outside unit tests. Base commit 2c05a38. Go 1.23 toolchain from /opt/go1.23, Claude Code 2.1.284, openspec 1.14.3, goreleaser v2.19.0 then v2.20.2 (`@latest`).

| # | Check | Result |
|---|-------|--------|
| 1 | Cross-compiled binaries (6 targets, vet windows/darwin) | PASS (built, not run on macOS/Windows: NOT POSSIBLE here) |
| 2 | goreleaser check + snapshot release | PASS (v2.19.0 and `@latest` v2.20.2) |
| 3 | Handoff concurrent with `git add` / `git commit` | PASS |
| 4 | Real openspec (`1/3 tasks`, `next:`, fallback) | PASS |
| 5 | Real Claude Code SessionStart hook | PASS (first run) |

## 1. Cross-compiled binaries

Platforms in `.goreleaser.yaml`: linux, darwin, windows x amd64, arm64 (6 pairs).

    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o /tmp/relai-verify-bin/relai-$os-$arch ./cmd/relai

All 6 built. `file` output: linux amd64/arm64 "ELF 64-bit LSB executable ... statically linked"; darwin amd64/arm64 "Mach-O 64-bit x86_64 / arm64 executable"; windows amd64/arm64 "PE32+ executable (console) x86-64 / Aarch64, for MS Windows".
`GOOS=windows go vet ./...` and `GOOS=darwin go vet ./...` both clean.

Caveat: building is not running. No macOS or Windows machine here, so runtime behaviour on those OSes is NOT verified.

Code review for Windows (static, not executed):
- `internal/initrepo/initrepo.go:64` `os.Symlink("CLAUDE.md", agents)`: on Windows without Developer Mode or admin rights this fails, and `relai init` aborts with "link AGENTS.md to CLAUDE.md" (returns the error, no fallback to a copy). Triggers when CLAUDE.md exists and AGENTS.md does not.
- `internal/scaffold/scaffold.go:115` `os.Symlink(link, target)`: same privilege issue when a template contains symlinks.
- `internal/initrepo/initrepo.go:109, 139, 186`: pre-commit, post-merge and session-start hooks are `#!/bin/sh` scripts. They work under Git for Windows (git runs hooks with its own sh), but `settings.json` line 207 runs `"$CLAUDE_PROJECT_DIR"/.claude/hooks/session-start.sh` directly, which needs a POSIX shell (Claude Code on Windows uses Git Bash, probably fine, unverified). `0o755` at lines 176 and 237 is a no-op on Windows; executability comes from git's shebang handling.
- Hooks rely on `mktemp`, `sed -E`, `[ -nt ]`, `timeout`: all in Git Bash, not in cmd/PowerShell.
- No `exec.Command("sh", ...)` anywhere in non-test Go code; all `exec.Command` calls use `git` directly (tidy.go:230,246, index.go:156,193, scan.go:190, links.go:144, workspace.go:64, handoff/util.go:18, initrepo.go:159, discover.go:296). No hard-coded "/" path joins found in non-test code except display strings (`discover.go:122,125` build slash command names, correct).

## 2. goreleaser

    goreleaser check                      -> 1 configuration file(s) validated
    goreleaser release --snapshot --clean --skip=publish

Ran first with the pre-installed v2.19.0, then installed `@latest` (v2.20.2, which auto-switched to the go1.27.4 toolchain; install succeeded) and repeated `check` and the snapshot release: both pass with 6 archives.
Result: "release succeeded after 6s". dist/ holds 4 `.tar.gz` (linux, darwin x amd64/arm64) + 2 `.zip` (windows) + `checksums.txt`; `sha256sum -c checksums.txt` reports all 6 OK. Extracted the linux_amd64 archive (contains `relai`, LICENSE, README.md): `./relai version` prints `relai 0.3.4-SNAPSHOT-2c05a38`, so the `-X main.version` ldflag works. `dist/` is in `.gitignore`; `git status` stays clean (the `go mod tidy` before-hook changed nothing).

## 3. Handoff concurrent with git add

Script: temp repo with 200 files + `relai init`, one commit.
- Idle: with a dirty working tree, `git status` once, then recorded `sha256sum` and mtime of `.git/index`, slept 1.1 s, ran `relai handoff` 10 times, compared: sha and mtime identical. INDEX UNCHANGED (handoff does not rewrite the index).
- Concurrent: background loop of 50 x `relai handoff >/dev/null` against a foreground loop of 50 x (append to a file; `git add -A`; `git commit -q -m x`).
  Observed: 0 git failures, git stderr empty (no `index.lock` message), handoff stderr empty with no non-zero exit, 51 commits, no leftover `.git/index.lock`.
Limit: the handoff runs are short and git ran 50 iterations; this shows no failure in this run, it is not a proof of absence of a race.

## 4. Real openspec

`openspec --version` 1.14.3. In a temp git repo: `openspec init --tools none --no-animation`, `openspec new change demo-change`, then wrote proposal.md and a tasks.md with `[x]` 1.1, `[ ]` 1.2, `[ ]` 1.3.
- `openspec list` printed `demo-change  1/3 tasks`.
- `relai handoff` section:

      ## Active changes (specs)
      - `demo-change` 1/3 tasks (in-progress)
        next: 1.2 Second open thing

- Fallback: with PATH = `/tmp/relai-verify-bin:/usr/bin:/bin` (no openspec):

      (the openspec command is not installed: changes below are read from `openspec/changes/*/tasks.md`)
      - `demo-change` 1/3 tasks (from files)
        next: 1.2 Second open thing

## 5. Real Claude Code SessionStart hook

Fresh temp git repo, freshly built relai first on PATH, `relai init`, commit, then added `- 2026-11-16 — the secret word is PAMPLEMOUSSE-42` under Decisions in `.relai/notes.md` (uncommitted). The hook script run by hand with `CLAUDE_PROJECT_DIR` set printed the line (handoff includes it).
Then ONE run from that directory:

    claude -p --model haiku --max-turns 2 "Without running any tool: what is the secret word in this project's decisions? Answer with the word only, or UNKNOWN."

Output: `PAMPLEMOUSSE-42`. The model had no tool use, so the word came from the SessionStart hook output. Folder trust did not block it in `-p` mode. 1 of 2 allowed `claude -p` runs used.

## Findings

1. Windows, `relai init` can fail: `internal/initrepo/initrepo.go:64`. Repro (on Windows without Developer Mode): a repo with CLAUDE.md and no AGENTS.md, run `relai init`; `os.Symlink` fails and init returns an error. Suggested: fall back to writing an AGENTS.md that points to CLAUDE.md. Not fixed: needs a design choice and cannot be tested here. Same class at `internal/scaffold/scaffold.go:115`.
2. Windows hooks assume a POSIX shell (`initrepo.go:109,139,186,207`). Works in Git Bash / Git for Windows, not elsewhere. Unverified; document it.
3. Note: `@latest` goreleaser now needs Go >= 1.27.4 (auto-downloaded toolchain). Not a relai bug; CI should pin a goreleaser version.

No bugs found in handoff concurrency, openspec parsing, goreleaser config or the SessionStart hook. No code changes were made.
