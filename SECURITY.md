# Security policy

Relai is a local command-line tool and a set of files in your repos. It has no server and listens on no
port. What it does touch: it runs `git`, `bd` and `openspec` in the directories you point it at, it can
`git clone` URLs you give it, and what it prints goes into the context of your AI agents. Please read
"Things to keep in mind" below.

(The optional team server, [RELAI-TEAM](https://github.com/ECG-Rudy-Glt/RELAI-TEAM), has its own
security policy.)

## Reporting a vulnerability

**Do not open a public issue with details.**

1. Use GitHub's private report: the repository's **Security** tab → **Report a vulnerability**.
2. If that button is not there, open a public issue titled `Security: please contact me` with **no
   details at all**; the maintainer will arrange a private channel.

Please include what you ran, what you expected and what happened, the version (`relai version`) and your
OS. This is a small project with one maintainer: expect an acknowledgement within about a week, a fix as
soon as one can be made safely, and credit in the release notes if you want it. There is no bug bounty.

## Supported versions

The project is v0.x. Only the latest release and `main` get fixes.

## Things to keep in mind

- **The handoff puts repo text in front of an AI agent.** `relai handoff` quotes commit subjects, task
  titles, file names and `.relai/notes.md`. Whoever can write those can write something that looks like an
  instruction ("prompt injection"). Agents should treat the handoff as information about the project, not
  as commands, and keep their own permissions tight. Equally, **do not put secrets in `notes.md`, task
  titles or commit messages**: the handoff sends them to whatever model your agent talks to.
- **Running Relai in a repo runs other tools there.** `relai handoff` and `relai discover` execute `git`,
  `bd` and `openspec` in that directory, and `relai index build` / `relai new` can clone URLs. Treat a repo you
  did not write like any code you are about to build: look before you run tools in it.
- **A template is code you will run later.** `relai new <template> <dir>` copies what the template
  contains, including scripts and CI files, and fills in `{{...}}` placeholders (a `text/template`
  with plain string values). Review a template you did not write before you use it.
- **A cloned repo can ship its own Claude Code hook.** Relai adds `.claude/hooks/session-start.sh`
  (it runs `relai handoff` when a session starts), and any repo can carry hooks. Claude Code asks you to
  trust a folder before running its hooks: read `.claude/settings.json` and the scripts it points to
  before you say yes to a repository you do not know.
- **The installer checks a checksum, not a signature.** `scripts/install.sh` downloads the archive and
  compares its SHA-256 with the release's `checksums.txt`, and stops if they differ or the checksum cannot
  be fetched. That catches a corrupted or swapped download. Both files come from the same release, so it
  does **not** protect against a compromised release or account. If that matters to you, build from source
  at a tag (`go install github.com/ECG-Rudy-Glt/RELAI/cmd/relai@v0.1.0`) or review the code first. Piping
  `curl` into `sh` is convenient, and you can download `scripts/install.sh` and read it first.
- **No network calls otherwise.** Besides cloning URLs you supply and the installer, Relai makes none, and
  it does not phone home.
- **`relai handoff` itself writes nothing to the repo it reads** (a guarantee of the format, see
  `spec/handoff.md` §6). The tools it runs are another matter: `bd` rewrites its own git-ignored database
  files under `.beads/` every time it runs, even to read.

## In scope

A way to make `relai` execute something the user did not ask for (including through a crafted repo,
template, `relai.yaml` or `.relai/index.json`), to read or write outside the directories given, to leak
secrets into the handoff or an error message, to bypass the installer's checksum check, or to overwrite a
file `relai init` / `relai new` promises not to overwrite.

## Out of scope

What a repo's own tools (`git`, `bd`, `openspec`) do when you run them there; social-engineering of an
agent through text you chose to put in your own repo; anything that needs write access to your machine;
reports against unsupported versions.
