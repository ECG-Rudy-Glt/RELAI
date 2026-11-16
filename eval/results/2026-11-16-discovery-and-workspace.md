# Measurements on real folders: `discover --links`, the `--propose` rule, `relai workspace`

Date: 2026-11-16. Tool: `relai` built from this branch (`go build ./cmd/relai`). Folders: the owner's real repositories
(read only; no file was written inside them). One person's repositories, one judge (an LLM reading the cited lines).
Repository names are replaced by placeholders (`proj-A`...); `proj-P`, `proj-Q` and `proj-R` stand for repositories whose
names are ordinary words, which matters for the results. Quoted file contents are paraphrased.

## Method

- A "mention" is one reported pair A -> B. It is **strong** when `strong_files > 0` (remote URL or relative path of B found in A),
  **weak** when only the name as a word was found (`strong_files == 0`).
- Each pair was judged by reading the cited lines in A (read only): **true** (A really works with, depends on, deploys or documents B as a
  project), **false**, or **unclear** (a pattern reference, a shared host name, an author link...).
- Precision is reported as true / (true + false), and strictly as true / all (unclear counted as not true).
- Runs: `discover --links <folder> [--depth 2] --json` on `~/projects/proj-C`, `/autres`, `/sites`, on `~/projects` (depth 2),
  and on `~` (depth 1). The projects and home runs overlap with the three sub-clusters (same repos), so "overall" below counts every
  distinct pair once.
- Evidence lists show up to three files; I opened those files and read the matching lines (and `grep`ed the rest of the repo when the lines were ambiguous).

## Part A: precision of strong and weak mentions

| Cluster | Strong | Strong true | Weak | Weak true / false / unclear | Weak precision (T/(T+F)) |
|---|---|---|---|---|---|
| projects/proj-C | 1 | 1 | 0 | - | - |
| projects/autres | 1 | 1 | 3 | 1 / 1 / 1 | 50% |
| projects/sites | 0 | - | 9 | 9 / 0 / 0 | 100% |
| projects (depth 2, all of the above plus cross pairs) | 5 | 5 | 27 | 22 / 1 / 4 | 96% |
| ~ (depth 1) | 25 | 25 | 45 | 19 / 18 / 8 | 51% |
| **Overall (distinct pairs)** | **28** | **28 (100%)** | **69** | **40 / 18 / 11** | **69% (strict 58%)** |

Overall = home (25 strong, 45 weak) plus the pairs that only exist in the projects run (3 strong, 24 weak: they involve the `*-site` /
`*-demo` repos or `proj-B-sync`, which the home scan does not list).

Strong mentions: 28 of 28 true. Every one was a clone URL, a link to the public repository, a deployment source or an `<owner>/<repo>` reference in a
tracker or doc. The only caveat: several repositories share a remote (`proj-E-oss` and `proj-E` on GitHub, `proj-B` and
`proj-B-sync`), so a strong mention of the public copy is also a mention of its private twin; I counted those as true.

### Weak false positives (18, all in the `~` run except one in `autres`)

| Pair | Reason |
|---|---|
| RELAI -> proj-E, proj-P, proj-C, proj-Q, proj-A, proj-F, proj-G, proj-R, proj-H, proj-B (10) | Self-reference: RELAI's own tasks, notes and specs quote the names of the repos it was tested on (and say "`proj-P` can be an ordinary word"). Not a dependency. |
| proj-P -> proj-Q | The name is an ordinary word, used as such in a sentence. |
| proj-Q -> proj-P | The name is an ordinary word, used as such in a sentence. |
| proj-A -> proj-P | The name is also a term of the project's own domain: it appears in an API route. |
| proj-A -> proj-Q | The name is an ordinary word, used as such in a to-do. |
| proj-E-oss -> proj-Q | Comment saying that containers named `proj-Q` etc. are deliberately absent. |
| proj-A -> proj-H | A product name inside advice strings (advice that names the product), not the repository. |
| proj-P -> proj-H | A generated graph file (`a-graph-file.html`) that contains proj-E node names. |
| proj-B -> proj-R | The name is an ordinary word, used as such in a sentence. |

Unclear (11): `proj-F -> proj-Q`, `proj-F -> proj-P`, `proj-A -> proj-F`, `proj-B -> proj-F`, `proj-G -> proj-P`,
`proj-G -> proj-F`, `proj-B -> proj-P`, `proj-A-oss -> proj-P`, `proj-B-sync -> proj-P`, `proj-C-demo -> proj-P`,
`proj-A-demo -> proj-P`. Causes: a shared path in a configuration file, a shared host name, "same pattern as
proj-P/proj-F" (an analogy), a palette "taken from the proj-P". A person could reasonably say yes or no.

Reading across the table: weak evidence is precise when the repository name is distinctive (`proj-C-demo`, `proj-A-demo`, `proj-D-demo`,
`proj-G`, `relai-team`: nearly all true) and poor when the name is an ordinary word (`proj-P`, `proj-Q`, `proj-R`),
or when the folder holds a repository about the tool itself (RELAI: 10 of the 18 false positives). Two repositories (`proj-P`, `proj-Q`) with
ordinary-word names produce 6 of the 8 non-RELAI false positives.

## Part B: precision of the `--propose` rule on a second real folder

`openspec/changes/auto-tidy-and-discovery-followups` says the rule was tuned on "the owner's folder, where documents-only mentions of a hub
repo were mostly notes and tests"; that is the hub repository proj-E in `~`, so I took **`~/projects --depth 2`** as the second
folder (14 repos, no `relai.yaml` in it, hub repos `proj-C` and `proj-P`). Judged against part A (27 true links exist in that folder: 5 strong,
22 weak).

| Setting | Entries proposed | true | false | unclear | Precision (T/(T+F)) | Recall of the 27 true links |
|---|---|---|---|---|---|---|
| default (`--min-weak 5`, or 2+ config files) | 6 | 6 | 0 | 0 | 100% | 6/27 = 22% (all 5 strong, 1 weak) |
| `--min-weak 2` | 24 | 23 | 0 | 1 (`proj-G -> proj-P`) | 100% (96% strict) | 23/27 = 85% |

What the default rule missed: 21 true weak links, all with fewer than 5 files and no CI/config file: the `*-site` -> `*-demo` pairs
(`proj-C-site -> proj-C-demo`, `proj-A-site -> proj-A-demo`, `proj-D-site -> proj-D-demo`), the footer links of every site to `proj-P`,
`proj-E-site -> proj-C-demo/proj-A-demo/proj-D-demo`, `proj-P -> proj-G/proj-C-demo/proj-D-demo/proj-A-demo`, and
`proj-D-demo -> proj-C-demo/proj-C` (engine reuse). With `--min-weak 2` only four 1-file links are missed (`proj-E-site -> proj-C-demo`,
`-> proj-A-demo`, `-> proj-D-demo`, `proj-D-site -> proj-D-demo`).

For reference, the same rule on `~ --depth 1` (the folder it was tuned on): default settings propose 41 entries, 37 true, 1 false
(`proj-P -> proj-Q`), 3 unclear: precision 97% (T/(T+F)), recall 37/44 = 84%. The 7 misses are all weak links with 2-3 files.

So on a folder whose mentions are mostly in prose in small repos, the default threshold is safe but very quiet (22% recall); lowering `--min-weak`
to 2 kept precision here but would not on the home folder, where 2-file weak mentions are mostly RELAI self-references (I did not run the
home folder with `--min-weak 2`).

## Part C: does `relai workspace` replace opening each project's handoff?

Setup: `relai index build --scan ~ --depth 1` found 14 repositories, but only **3 have a `relai.yaml`** (relai, relai-team, proj-E);
11 were skipped, so the overview (1,442 bytes, ~7.6k tokens of prompt including Claude Code's fixed overhead) covers 3 projects. Questions were
therefore asked about "the indexed projects / projects with a `relai.yaml`", with the answers checked against the files:

1. Which project has the most open tasks, and what is its most urgent task? Truth: proj-E; P0 tasks `hl-2y1` and `hl-58` (either accepted).
2. Which project provides `team.board`, and what is its first listed open task? Truth: relai-team, `RELAI-TEAM-2k3` (first in its `provides` and task list).
3. Which project has the oldest last commit, and what date? Truth: proj-E, 2026-11-14.

Condition (a): prompt = content of `/tmp/ws.md` + question, no tools. Condition (b): question plus "the projects are the git repositories under ~
that have a relai.yaml; you may run `relai handoff <dir>` and `ls`", `--allowedTools "Bash(relai handoff:*) Bash(ls:*) Read"`.
All runs: `claude -p --model haiku --output-format json --max-turns 15 --setting-sources project --strict-mcp-config`, from `~`, 6 runs in total.

| Q | Condition | Correct | num_turns | total_cost_usd | Note |
|---|---|---|---|---|---|
| 1 | (a) overview | yes (proj-E, `hl-2y1`) | 1 | 0.0187 | |
| 2 | (a) overview | yes (relai-team, `RELAI-TEAM-2k3`) | 1 | 0.0068 | |
| 3 | (a) overview | yes (proj-E, 2026-11-14) | 1 | 0.0060 | |
| 1 | (b) handoffs | no answer | 3 | 0.0303 | stopped to ask for permission |
| 2 | (b) handoffs | yes | 15 (the cap) | 0.0826 | reached the answer on the last turn; 4 denied commands |
| 3 | (b) handoffs | no answer | 4 | 0.0260 | stopped to ask for permission |

Totals: (a) 3/3 correct, 3 turns, 0.032 USD; (b) 1/3 correct, 22 turns, 0.139 USD.

Honest reading of (b): the model kept issuing `rtk find ...` / `rtk ls -la ...` (a command prefix coming from the owner's global instructions) and
`for` loops, none of which match the allowed patterns `Bash(relai handoff:*)`, `Bash(ls:*)`; so 8 commands were denied over the three runs and two runs
gave up. That is a permission-pattern artifact of my setup, not a limit of the handoff. I could not rerun (6-run cap). What (b) does show is how
much discovery work an agent must do before it even reaches a handoff: finding which of 14 directories hold a project (3 turns minimum, 15 in the one run that
finished), against one turn when the overview is in the prompt.

## What this means

- Strong mentions looked reliable here (28 of 28). Weak mentions are a different tool: about 70% precise overall, from 100% (distinctive names in
  small repos) to about 50% (ordinary-word names such as `proj-P`/`proj-Q`, and the repo about relai itself). They are fine as evidence for a
  person or an agent to confirm, as the feature states, and should not be written without that check.
- The `--propose` rule never proposed a false link on `projects` and one on `home`, which supports the thresholds; the cost is recall (22% there with
  the default). A lower `--min-weak` helps on small distinctive-name repos but was not tested where it would hurt.
- The workspace overview answered three simple lookup questions in one turn for 0.006-0.019 USD each. It supports "replaces opening each handoff"
  only for questions the overview itself contains (counts, the top three tasks, provides, last commit); it does not show that for questions that need
  a project's notes or other tasks. The comparison with (b) is weak because two (b) runs were blocked by permission patterns.
- Limits: one owner's repositories, most of them French-language documentation and small (n = 97 mentions, 3 questions, 6 runs); one judge, an LLM, who
  decided "unclear" cases by reading three evidence files at most; the overview covered only 3 of 14 repositories because the rest have no `relai.yaml`
  (so the real work needed before `workspace` is useful is `relai init`, or reading the `--scan` skipped list); task counts in the overview lag the live
  Beads database (57 open for proj-E against 58 open in the current export), as the output itself says.
