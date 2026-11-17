# Interrupted-work measure with Codex — instructions (round 2)

**What changed since round 1.** Round 1 (the multi-repository fixtures of `eval/multi`) gave, on the hard variant,
1/3 with Relai against 2/3 without: in both failures Codex read the line saying a linked project had moved and still
used the old API. Relai now (handoff format **v0.10**) gives an instruction in that line and shows what changed: the
new upstream commit subjects, the upstream dated notes, the command to read them. Round 2 measures that, on **new
tasks** written by a separate agent that had no access to Relai's code or results (`eval/resume2/TASKS.md` describes
them). Do not use `eval/multi` again: its hard variant now has the answer in the line.

**The question.** When an agent picks up work another session left unfinished, does giving it `relai handoff` make it
finish correctly more often, or for fewer tokens? Each task has a fact, written in the repository state (notes,
commit messages, a linked repository), that the obvious continuation gets wrong. Hidden tests grade the final code.

## Setup (once)

1. Codex CLI logged in (note the version), `git`, `jq`, Python 3.10+.
2. Relai **v0.10 handoff**: until a release has it, build from the branch:
   `git clone https://github.com/ECG-Rudy-Glt/RELAI && cd RELAI && git checkout feat/upstream-subjects && go build -o relai ./cmd/relai`
   (or `main` once PR #34 is merged). Check: `./relai version`, and that `RELAI/eval/resume2/build.py` exists.
3. Your Windows launcher from round 1 (PowerShell) works the same way; port these differences from
   `eval/resume/run-codex.sh`: the agent **edits files**, so the sandbox is `workspace-write` (not read-only); the
   tasks come from `FIXTURES=eval/resume2`; a task whose build makes an `app` folder is multi-repository: run the
   agent in `app` (the linked repositories are next to it, in the parent folder; if Codex cannot read them, add the
   parent with `--add-dir ..` and say so); grading is `python3 hidden.py <task> <dir>` on the whole built folder,
   last line `PASS` or `FAIL: …`. Keep the same model and settings for every run.

## Run

Tasks: `s1 s2 s3` (one repository) and `m1 m2 m3` (several linked repositories). Conditions: `repo` (just the
instruction; the notes are in the repository and the agent may read them) and `relai` (the same plus the output of
`relai handoff`). **5 runs per cell: 6 tasks × 2 conditions × 5 = 60 runs.** Pilot first: one `relai` run of `m1`,
read the agent's answer and the verdict, check the token counters against the events.

```sh
export RELAI_BIN=$PWD/relai FIXTURES=$PWD/eval/resume2 CODEX_MODEL=<your model>
for t in s1 s2 s3 m1 m2 m3; do for c in repo relai; do for n in 1 2 3 4 5; do
  sh eval/resume/run-codex.sh $c $t $n results-codex-resume.jsonl
done; done; done
jq -rs 'group_by(.cond)|.[]|"\(.[0].cond): \(map(select(.pass))|length)/\(length) passed, commands \(((map(.commands)|add)/length)*10|floor/10), input \(((map(.input)|add)/length)|floor)"' results-codex-resume.jsonl
jq -rs 'group_by(.task+.cond)|.[]|"\(.[0].task) \(.[0].cond): \(map(select(.pass))|length)/\(length)"' results-codex-resume.jsonl
```

## Send back

`results-codex-resume.jsonl`, the two summaries, Codex version, model and reasoning setting, the OS, your launcher if
it differs, any run you excluded and why. Do not change the tasks, the hidden tests or the grading. Keep the raw
events privately (they can contain your Codex profile); they are only needed if a result looks wrong.
