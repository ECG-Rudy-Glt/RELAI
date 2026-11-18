# RELAI — reprise de travail interrompu avec Codex, round 2

Série arrêtée à la demande de l’utilisateur après **49 mesures terminées sur 60**. Aucun processus de mesure ne tourne encore. Les résultats sont partiels ; les deux conditions n’ont pas exactement le même nombre de mesures.

| Condition | Réussites | Commandes moyennes | Jetons d’entrée moyens |
|---|---:|---:|---:|
| repo | 23/25 (92 %) | 6,08 | 119 087 |
| relai | 21/24 (87,5 %) | 6,29 | 129 802 |

Dans ces résultats partiels, RELAI ne montre pas de gain de réussite ni d’économie de jetons. L’entrée moyenne est environ 9 % plus élevée avec RELAI. Cette observation ne constitue pas une conclusion générale : échantillon limité, série interrompue et effectifs déséquilibrés. Les résumés texte appliquent exactement les arrondis demandés : commandes tronquées à une décimale, jetons à l’entier inférieur.

| Tâche | repo | relai |
|---|---:|---:|
| s1 | 5/5 | 5/5 |
| s2 | 5/5 | 5/5 |
| s3 | 5/5 | 5/5 |
| m1 | 4/5 | 3/5 |
| m2 | 4/5 | 3/4 |
| m3 | Non lancée | Non lancée |

Les échecs m1 cherchent le fichier à l’ancien emplacement plat, au lieu de son emplacement par préfixe. Les échecs m2 ignorent `SVC_WORKERS`. Les cinq échecs ont été conservés, sans changer le correcteur ni relancer les mesures.

## Environnement

- Codex CLI autonome officielle : **0.169.0**.
- Modèle : **gpt-6.1-sol** ; raisonnement : **medium**, constant dans toutes les exécutions.
- Windows natif, noyau **10.0.26200.0**, PowerShell **7.6.9**.
- Python **3.13.15**, Git **2.56.0.windows.4**.
- RELAI compilé avec Go **1.23.12**, `GOTOOLCHAIN=local`.
- Branche distante `feat/upstream-subjects`, commit **3bf2b9d7665d5135fa9f5020b0be967ab1d821c3** ; branche locale `round2-upstream`.
- `relai version` retourne `relai dev`. Le handoff du pilote a confirmé explicitement **format v0.10**, avec les sujets de commits amont, notes datées et commandes de lecture.

## Protocole et adaptations

Seules les nouvelles fixtures **eval/resume2** ont été utilisées. Aucun test du round 1 n’a été relancé. L’ordre séquentiel est celui demandé : s1, s2, s3, m1, m2, m3 ; repo puis relai ; cinq répétitions par cellule. Chaque exécution part d’une fixture neuve et d’une session Codex éphémère.

Le lanceur PowerShell livré adapte `eval/resume/run-codex.sh` :

- Construction par `build.py` original ; notation par `hidden.py` original sur le dossier construit entier. La dernière ligne détermine exactement `PASS` ou `FAIL: …`.
- Prompt de base et ajout de `relai handoff` conservés. Les notes restent accessibles dans la condition repo.
- Bac à sable **workspace-write**. Pour m1/m2, Codex démarre dans **app**, avec `--add-dir <parent construit>`, équivalent absolu de `--add-dir ..`. Cette même règle était prévue pour m3.
- Compteurs additionnés depuis `turn.completed.usage` ; commandes dédupliquées par identifiant dans les événements `item.completed` de type `command_execution`.
- `cached_input` est conservé séparément : il n’est pas ajouté une deuxième fois à `input`. Les jetons affichés incluent la part en cache et ne constituent pas un coût monétaire.
- Configuration de la CLI autonome réutilisée du round 1 : `--ignore-user-config`, `--ephemeral`, `--disable apps`, `windows.sandbox="elevated"`. Le profil habituel et ses instructions utilisateur contribuent à la surcharge du modèle.
- Variables d’intégration Desktop retirées du processus lanceur : `CODEX_WINDOWS_REGISTERED_CORE`, `CODEX_APP_RUNTIME_CLI_PATH`, `CODEX_APP_TOOLS_PIPE_PATH`. Les restrictions de permission sont conservées ; aucun mode sans bac à sable n’est utilisé.
- Nouveau binaire RELAI dans un dossier dédié ajouté au PATH de toutes les conditions ; l’ancien binaire du round 1 n’est pas utilisé.
- Python appelé par son chemin absolu. `PYTHONUTF8=1` pour toutes les mesures principales afin de conserver l’encodage des notes sous Windows. Cet ajustement d’environnement a été ajouté après le pilote ; aucune logique de tâche ou de test n’a été modifiée.

## Pilote et mesures non incluses

Le pilote **m1 / relai / run 0** a réussi : `PASS`, **8 commandes**, **170 952** jetons d’entrée dont **146 560** en cache, **1 135** en sortie. La réponse finale indique l’ajout de `title_of`, l’adaptation au stockage par préfixe et les tests réussis. Réponse et compteurs ont été contrôlés dans les événements. Le pilote est séparé des mesures principales, conformément au protocole.

Le premier contrôle automatique d’autorisation du pilote avait expiré avant tout lancement. La relance unique autorisée a réussi ; cette expiration n’a produit aucune exécution de modèle ni ligne de mesure.

À la demande d’arrêt :

- **m2 / relai / run 5** était en cours. Son processus a été interrompu ; aucun verdict n’a été enregistré. Il n’est compté ni comme réussite ni comme échec.
- Les **dix mesures m3** n’ont pas été lancées.
- Les **49 mesures terminées** sont toutes conservées. Aucune n’a été remplacée ou exclue selon son résultat.

## Vérifications et confidentialité

Les 49 verdicts et tous leurs compteurs ont été revérifiés contre les fichiers privés. Les empreintes SHA-256 de `build.py`, `hidden.py` et du binaire RELAI sont identiques avant et après la série. Le dépôt source RELAI est inchangé. Les deux résumés correspondent aux calculs jq demandés, appliqués aux résultats disponibles.

La série a été arrêtée par interruption de son processus. Un contrôle des processus a ensuite confirmé l’absence du lanceur et des sessions de mesure actives.

Les événements, prompts, réponses complètes et dossiers de travail restent privés sous `work/resume2-private` et `work/ws-resume-*`. **Ils ne figurent pas dans les livrables ni dans l’archive.** Aucun fichier du profil Codex n’est livré.

## Fichiers

- `results-codex-resume.jsonl` : 49 résultats terminés.
- `summary-conditions.txt`, `summary-tasks.txt` : les deux résumés demandés.
- `pilot-codex-resume.jsonl` : pilote séparé.
- `summary.json`, `verification.json` : agrégats et audit sans événements bruts.
- `run-resume2.ps1`, `run-resume2-suite.ps1` : lanceurs utilisés. Ils supposent leur emplacement sous `work/`, le clone sous `work/RELAI`, le binaire sous `work/round2-bin/relai.exe`, la CLI sous `work/tools/codex-cli` et les résultats sous `outputs/round2`. Le chemin absolu de Python doit être adapté sur une autre machine.

Le lanceur de série fourni décrit toujours les 60 mesures prévues ; il ne faut pas le relancer sur les dossiers existants pour continuer automatiquement cette série arrêtée.
