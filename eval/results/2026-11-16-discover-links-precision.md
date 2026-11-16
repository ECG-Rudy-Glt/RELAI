# Measurements: making the weak mentions of `discover --links` more precise (task relai-p30)

Date: 2026-11-16. Baseline: `relai` at origin/main (commit 0267b05). Folders: the owner's real repositories, read only, same two runs and
same labels as `2026-11-16-discovery-and-workspace.md` (placeholders `proj-A`... as there; `proj-P`, `proj-Q`, `proj-R` are repositories whose
names are ordinary words). One owner, 14 repositories, **tuned and judged on the same data** (see Limits).

## Method

- Runs: `discover --links ~ --json` ("home", 14 repositories, 25 strong and 45 weak mentions) and `discover --links ~/projects --depth 2 --json`
  ("projects", 14 repositories, 5 strong and 27 weak). The baseline reproduced the earlier counts exactly (25/45 and 5/27), so the
  earlier labels apply pair by pair: strong = true (28 of 28); weak pairs listed there as false or unclear keep that label; every other weak pair is true.
  Home weak: 19 true, 18 false, 8 unclear. Projects weak: 22 true, 1 false, 4 unclear. I did not re-judge any pair; the variants only
  remove files from the evidence, so no new pair appears.
- "Weak precision" = true / (true + false). "Propose" is the rule of `--propose`: strong evidence, or 2+ CI/deploy/config files, or at least `--min-weak`
  files. Propose precision is true / (true + false) of the proposed lines; recall is the proposed true lines over the true links (home 44, projects 27).
- Each idea was run alone, then combined. The variants are `Options` of `scan.LinksWith`; the shipped values are `scan.DefaultOptions()`.
  "Ordinary word" = the folder name is letters `a-z` only (no capital, digit, dash, underscore or dot) and at most N letters; its weak mention needs
  `/name` followed by a non-name character, `name/`, or `name.git` (case-insensitive) instead of the bare word.

## Results: weak mentions (true / false / unclear)

| Variant | Home | Home precision | Projects | Projects precision |
|---|---|---|---|---|
| Baseline | 19 / 18 / 8 | 51% | 22 / 1 / 4 | 96% |
| 1. Skip own task/notes/spec/handoff files | 18 / 7 / 8 | 72% | 22 / 1 / 4 | 96% |
| 2. Ordinary word, N = 8 | 19 / 17 / 8 | 53% | 22 / 1 / 4 | 96% |
| 2. Ordinary word, N = 10, 12 or no limit | 18 / 7 / 6 | 72% | 22 / 0 / 1 | 100% |
| **1 + 2 (N = 12), shipped** | **17 / 2 / 6** | **89%** | **22 / 0 / 1** | **100%** |

Idea 1 alone removes 11 false mentions (10 are RELAI quoting the repositories it was tested on) and loses one true link, `proj-E -> RELAI-TEAM`, whose
only evidence was in `.beads/` and `openspec/`. Idea 2 alone removes 11 false mentions and 1 true one (`proj-B -> proj-A`, bare word in a plan file),
and 2 unclear ones. They overlap partly, so combined they remove 16 false, 2 true, 2 unclear. The two false mentions left are
`proj-A -> proj-P` (the name is an API route, so it is path-like) and `proj-B -> proj-R` (a plan file).

The length limit N matters: 8 (the example in the task) catches only one false mention and no true one, because the ordinary-word names here have 9 to 10 letters
(`proj-P`, `proj-Q`, `proj-A`, `proj-H`). N = 10, 12 and no limit give identical results on these 14 repositories; I shipped 12 only so that a long coined
name is not treated as a word. That choice is not supported by data.

## Results: `--propose`

| Variant | `--min-weak` | Home: proposed / T / F / U, precision, recall of 44 | Projects: proposed / T / F / U, precision, recall of 27 |
|---|---|---|---|
| Baseline (before) | 5 | 41 / 37 / 1 / 3, 97%, 84% | 6 / 6 / 0 / 0, 100%, 22% |
| Baseline | 2 | 58 / 41 / 10 / 7, 80%, 93% | 24 / 23 / 0 / 1, 100%, 85% |
| Idea 1 | 5 | 40 / 36 / 1 / 3, 97%, 82% | 6 / 6 / 0 / 0, 100%, 22% |
| Idea 1 | 2 | 50 / 40 / 3 / 7, 93%, 91% | 24 / 23 / 0 / 1, 100%, 85% |
| Idea 2 (N >= 10) | 2 | 49 / 40 / 4 / 5, 91%, 91% | 21 / 20 / 0 / 1, 100%, 74% |
| 1 + 2 | 5 | 37 / 35 / 0 / 2, 100%, 80% | 5 / 5 / 0 / 0, 100%, 19% |
| **1 + 2, `--min-weak 2` (shipped)** | **2** | **44 / 38 / 1 / 5, 97%, 86%** | **21 / 20 / 0 / 1, 100%, 74%** |

Over the two folders (71 true links): before, 43 true of 44 proposed with a false one, recall 61%, precision 98%; after, 58 true of 59 proposed,
recall 82%, precision 98%. `--min-weak 2` alone on the old code would have given recall 90% but precision 86% (10 false lines on home, as the earlier
report suspected); the precision rules are what make the lower default safe. The trade: recall on the projects folder (many small sites whose
only link is a bare `portfolio` word or a footer) drops from 85% to 74% against the old rule at 2, but is 74% against 22% for the old default.

## Shipped and rejected

- Shipped: idea 1 (own task, notes, spec and handoff files are not weak evidence; strong evidence in them still counts). Files: `.relai/`, `.beads/`,
  `openspec/`, and any file named `*todo*`, `*tasks*`, `*handoff*`, `*passation*` (case-insensitive). Cost: one true link.
- Shipped: idea 2 with N = 12. Cost: two true links and some counts (`proj-B -> proj-A`, `proj-E -> proj-P` from 41 files to 11); the footer links to `proj-P` of the sites
  stay (they are URLs).
- Shipped: `--min-weak` default 2 (idea 3), only together with 1 and 2.
- Idea 4 (one kind of name): the code already printed the relai.yaml id when the target has one and its folder name otherwise, which is the mix the report saw. A single kind is not
  possible (a repository without relai.yaml has no id, and `links:` wants ids), so the line now says `# folder name, no relai.yaml yet` when it prints a folder name.
  Not measured: a presentation change.
- Rejected: N = 8 for the ordinary-word rule (no effect). Not tried: a dictionary or a longer word list (the task forbids it).

## What a user of the previous version notices

Fewer weak mentions (home: 45 to about 25 pairs), weak counts that shrink for names like `portfolio`, `--propose` proposing more lines by default (`--min-weak` 2 instead of 5),
and the `; folder name, no relai.yaml yet` note. A repository whose only mention of another is in its own TODO, tasks or `openspec/` files is no longer reported unless it is strong.

## Limits

- One owner, two folders that overlap heavily (14 repositories), 71 true links, 19 weak false mentions before. The rules were chosen on this data and measured on it: the
  gain is not a held-out result. The ordinary-word names here are 4 repositories; a folder of other names may behave differently (a distinctive single word such as a product
  name loses its bare-word mentions).
- The labels come from the earlier report (one LLM judge, 11 "unclear" pairs not counted as false); I did not re-judge them. Unclear pairs are listed separately.
- Not measured: an agent following `--prompt`; the effect on a folder with no RELAI-like repository.
