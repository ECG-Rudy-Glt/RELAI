# Post LinkedIn — Relai (prêt, à publier le mercredi 18 novembre 2026)

Mis à jour le 2026-11-16 pour Relai v0.9.0. Chiffres : `eval/README.md` (4 projets de test, Claude Haiku et Sonnet).
Comparaison avec les autres outils : https://relai.gault-rudy.com/fr/docs/compare/

## Version française

```
Votre agent IA relit tout votre dépôt à chaque session. Il n'y est pas obligé.

À chaque démarrage, un agent redécouvre ce qu'un collègue demanderait en cinq minutes : où en est le projet, quelle est la suite, ce qui a déjà cassé. Des milliers de tokens relus, et parfois une vieille liste de tâches prise pour la vérité.

J'ai construit Relai pour cela : des fichiers dans git et un binaire Go. Pas de serveur, pas de compte, aucun appel à un modèle.

→ relai handoff : l'état du projet en 2 000 tokens au plus, calculé depuis git, vos tâches (Beads, GitHub ou GitLab Issues), vos spécifications OpenSpec et un fichier de notes
→ relai find : savoir avant de coder si un autre de vos projets fournit déjà la fonction
→ relai sync et relai check : les mêmes règles dans tous vos dépôts, à une version figée, vérifiées en CI
→ relai guide --install : n'importe quel agent sait utiliser l'outil, dans n'importe quel projet

Pour les équipes, Relai-Team (facultatif, auto-hébergé) ajoute un tableau commun où chaque personne ou agent réserve une tâche, pour que deux ne démarrent jamais la même.

Pas de chiffre ronflant : mes premiers essais portaient sur de petits projets que j'ai écrits moi-même, ce n'est pas une preuve. Ce que je sais : le résumé n'apporte rien sur un dépôt sans suivi de tâches ni notes, car Relai résume ce qui existe, il n'invente rien. Je teste maintenant sur des dépôts plus grands et plus complexes.

Ce n'est pas un outil de mémoire comme mem0 ou claude-mem, ni une méthode comme Trellis ou OpenSpec. Il calcule un résumé, toujours le même pour un même état du dépôt. La comparaison honnête est sur le site.

Version 0.x, Apache 2.0, un seul développeur derrière, testé avec des modèles Claude uniquement. Je cherche surtout les cas où ça casse.

Site : https://relai.gault-rudy.com
Code : https://github.com/ECG-Rudy-Glt/RELAI

#OpenSource #IA #AgentsIA #DevTools
```

## English version

```
Your AI agent re-reads your whole repo every session. It doesn't have to.

At every start, an agent re-discovers what a colleague would ask in the first five minutes: where is the project, what's next, what already broke. Thousands of tokens re-read, and sometimes a stale task list taken for the truth.

So I built Relai: files in git plus one Go binary. No server, no account, no model call.

→ relai handoff: the project state in at most 2,000 tokens, computed from git, your tasks (Beads, GitHub or GitLab Issues), your OpenSpec specs and a notes file
→ relai find: check whether another of your projects already provides it before you build it
→ relai sync and relai check: the same rules in all your repos, pinned to a version, checked in CI
→ relai guide --install: any agent knows the tool, in any project

For teams, Relai-Team (optional, self-hosted) adds a shared board where each person or agent claims a task, so two never start the same one.

No big numbers: my first trials used small projects I wrote myself, which proves little. What I do know: the summary adds nothing on a repo with no task tracker or notes, because Relai summarises what exists, it invents nothing. I am now testing on larger, more complex repos.

It is not a memory tool like mem0 or claude-mem, nor a method like Trellis or OpenSpec. It computes a summary, always the same for the same repository state. The honest comparison is on the site.

v0.x, Apache 2.0, one developer, tested with Claude models only. I mostly want to hear where it breaks.

Site: https://relai.gault-rudy.com
Code: https://github.com/ECG-Rudy-Glt/RELAI

#OpenSource #AI #AIAgents #DevTools
```

## À vérifier avant de publier

- Le post ne cite volontairement aucun chiffre de gain : les mesures de `eval/README.md` portent sur 4 petits projets écrits par l'auteur (2 à 3 essais, modèles Claude seulement) et un test sur 27 dépôts réels n'est pas terminé (24 sessions sur 108, sans réponse notée). Ne pas ajouter de chiffre avant un test plus large.
- GitLab Issues (`tasks: gitlab`, via `glab`) n'a jamais été lancé contre un vrai GitLab : ne pas le présenter comme éprouvé.
- Versions publiées : Relai v0.9.0, Relai-Team v0.1.2.
