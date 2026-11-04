# Contributing to Relai

Thanks for helping. Relai is a small project run by one maintainer, built so that humans **and** AI agents
can pass the baton between sessions and across repos. The rules below are the same for both.

## Ways to help

- **Report a bug or a rough edge**: open an issue with the command you ran, what you expected, what
  happened, and `relai version`.
- **Report a security problem**: not as a public issue, see [SECURITY.md](SECURITY.md).
- **Fix the docs**: a README or spec that disagrees with the behaviour is a bug.
- **Try it on a real project** (especially one that is not a Go project, or has no Beads/OpenSpec) and tell us
  where the handoff was wrong, too long, or missed something. Real repos find what fixtures do not.
- **Pick up a task**: see "Picking a task" below.
- **Implement the handoff format in another tool**: the format is specified in [`spec/handoff.md`](spec/handoff.md).

## Set up

You need Go 1.21+ (the module pins toolchain 1.26.10, which Go downloads by itself) and `git`. `bd` ([Beads](https://github.com/steveyegge/beads)) and `openspec` (Node
22.12+) are optional: without them the code paths that use them are covered by tests with fake commands, but
you will want them to try the real thing.

```sh
git clone https://github.com/ECG-Rudy-Glt/RELAI && cd RELAI
bd bootstrap --yes                 # loads the task list from .beads/issues.jsonl (a fresh clone has none)
go build -o relai ./cmd/relai && ./relai handoff
go test -race ./...
```

On Debian/Ubuntu the packaged Go may be older than 1.21 and unable to fetch the pinned toolchain: install a current
one rather than lowering `go.mod`.

If you use Claude Code, `CLAUDE.md` and a session-start hook load the project's handoff for you; other agents
start from `AGENTS.md`.

## Before you open a pull request

```sh
gofmt -l .            # must print nothing
go vet ./...
go test -race ./...
```

- **Standard library only** unless a dependency is clearly worth it. Relai is one static binary.
- **It must work for one person with only git.** A feature that needs a server or an account does not belong
  here (that is what [RELAI-TEAM](https://github.com/ECG-Rudy-Glt/RELAI-TEAM) is for).
- **Tests come with the change**, and they use real `git` repos in a temp dir rather than mocks of git.
  Back-date fixture commits with `GIT_AUTHOR_DATE` / `GIT_COMMITTER_DATE` when time matters.
- **The handoff is deterministic and never calls an LLM.** The same repo gives the same bytes, the token budget
  is met by showing fewer items rather than rewording, and Relai itself must write nothing while building a handoff (the tools it runs, like `bd`, keep their own
  git-ignored files).
- **Changing the handoff's shape changes its version.** A new section or line kind is a minor bump in
  `FormatVersion`, `spec/handoff.md` (structure, cut order, changelog) and the tests, together.
- **Everything is in English**: code, docs, commits, issues.
- **Commit messages** are conventional (`feat:`, `fix:`, `docs:`, `test:`, `chore:`) with no
  `Co-Authored-By` or "Generated with" trailers (a project rule, also for AI-written commits).
- **Keep the README true.** If a command, flag or limit changes, change the README in the same PR.
- **One concern per PR**, with a description that says what changed, why, and how you verified it.

### Larger changes go through OpenSpec

A new feature or a change in behaviour starts as a change under `openspec/changes/<name>/`: a `proposal.md`
(why), `design.md` (decisions and trade-offs), spec deltas under `specs/` (what must be true, as WHEN/THEN
scenarios) and `tasks.md` (small steps, each with how to verify it). Validate with `openspec validate
<name>`; when the work is done, `openspec archive <name>` merges the deltas into `openspec/specs/`. Small
fixes and doc changes do not need one. If unsure, open an issue first.

### Picking a task

Tasks live in Beads, not in GitHub issues: `bd ready` lists what is unblocked, `bd show <id>` gives the detail.
The task list reaches other people through `.beads/issues.jsonl`, so after creating, updating or closing a task
run `scripts/sync-beads.sh` and commit that file (it removes the personal owner field).

Every `bd` run rewrites about 3 MiB of its database, even for a read: never run it in a loop or on a timer,
and keep the number of `bd` calls per `relai handoff` as low as it is.

## Be careful with the evaluation harness

`go run ./eval/e2` starts real agents (`claude -p`) and **uses your quota**. The unit tests do not. Run it only
when you changed what the handoff says, and read `eval/README.md` first.

## For AI agents

Start with `AGENTS.md`. Record decisions and pitfalls in `.relai/notes.md`, not in chat. Test with real
processes where it matters, say what you did **not** verify, and never publish, merge or tag without being
asked.

## Licence

By contributing you agree that your contribution is licensed under the [Apache License 2.0](LICENSE), the same
as the project (section 5 of the licence).
