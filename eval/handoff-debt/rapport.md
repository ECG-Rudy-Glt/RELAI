# Mesure RELAI avec Codex — 17 novembre 2026

Les 18 mesures prévues sont terminées. Sur la variante facile, toutes les conditions réussissent. Sur la variante difficile, RELAI réussit 1/3, contre 2/3 sans aide et 2/3 avec documentation. Cette petite série ne confirme donc pas un avantage de RELAI à qualité égale pour un second fournisseur.

| Variante | Condition | Correct | Commandes moyennes | Entrée moyenne | Dont cache |
|---|---|---:|---:|---:|---:|
| easy | none | 3/3 | 4.67 | 65642 | 60330 |
| easy | docs | 3/3 | 4.33 | 81147 | 73130 |
| easy | relai | 3/3 | 4.33 | 75986 | 66517 |
| hard | none | 2/3 | 5.67 | 89308 | 79488 |
| hard | docs | 2/3 | 6.33 | 111029 | 100565 |
| hard | relai | 1/3 | 5.00 | 78974 | 70656 |

Les jetons sont arrondis à l'entier inférieur, conformément au résumé demandé. Les commandes de ce tableau sont des moyennes à deux décimales ; `summary.txt` applique aussi l'arrondi inférieur aux commandes, comme la commande jq fournie.

Par rapport à `none`, RELAI utilise environ 15,8 % de jetons d'entrée supplémentaires en facile et 11,6 % de moins en difficile. Les commandes baissent respectivement de 7,1 % et 11,8 %. En difficile, cette baisse accompagne davantage d'erreurs : elle ne démontre pas une meilleure efficacité. Trois répétitions par condition sont insuffisantes pour une conclusion générale. Aucun rapprochement avec les chiffres Claude n'est effectué.

## Environnement

- Dépôt : https://github.com/ECG-Rudy-Glt/RELAI ; branche `feat/workspace-handoff`.
- Commit : `b34ea0d5dadc322394ba91e4a7deed52bd4b7aa0`.
- CLI mesurée : `codex-cli 0.169.0`, distribution npm officielle installée localement.
- CLI Desktop initiale : `codex-cli 0.169.0-alpha.11.1` ; essais de diagnostic uniquement, exclus des 18 mesures.
- Modèle constant : `gpt-6.1-sol`, raisonnement `medium`.
- Windows, PowerShell 7.6.9, Git for Windows ; RELAI compilé avec Go 1.23.12 et `GOTOOLCHAIN=local`. Le module déclare Go 1.23, mais propose Go 1.26.10 via sa directive toolchain ; cette sélection automatique n'a pas été utilisée.
- Ordre : easy puis hard ; none, docs, relai ; trois répétitions séquentielles. Chaque mesure utilise une nouvelle fixture et une session éphémère.

## Adaptations du lanceur

`run-codex.sh` dans le dépôt n'a pas été modifié. L'adaptation Windows est fournie dans `run-codex.ps1` et `run-suite.ps1`. La fixture est créée directement par le `build.sh` original avec le shell Git for Windows. Le barème reste inchangé : la dernière ligne commençant par `NEXT:` doit contenir `wa-3` ou `invoice`, ainsi que `v2` ou `cursor`.

Les changements techniques sont :

1. Remplacement de jq et du lanceur shell par PowerShell pour lire les événements et écrire le JSONL.
2. Comptage des identifiants uniques de commandes `item.completed`, au lieu de compter aussi `item.started`. Vérification visuelle : quatre commandes produisent huit événements dans le pilote et dans la première mesure sans aide.
3. Extraction des compteurs de `turn.completed.usage`. `input_tokens` est repris tel quel ; `cached_input_tokens` est conservé séparément, sans l'ajouter une deuxième fois. Les 18 sessions ont chacune un seul événement `turn.completed` : cette valeur ne mesure pas le nombre d'appels internes du modèle.
4. Conservation des prompts, réponses finales, événements JSONL et stderr ; refus d'enregistrer comme mesure valide un échec CLI ou une session sans commande locale terminée.
5. `--ignore-user-config`, `--ephemeral`, `--disable apps` et sélection explicite `windows.sandbox="elevated"`, avec `--sandbox read-only` maintenu. Le profil Codex habituel et ses instructions utilisateur restent utilisés par la CLI autonome ; leur surcharge fait partie des compteurs de cette série.
6. Pour la CLI autonome uniquement, suppression dans le processus lanceur de trois variables d'intégration Desktop : `CODEX_WINDOWS_REGISTERED_CORE`, `CODEX_APP_RUNTIME_CLI_PATH`, `CODEX_APP_TOOLS_PIPE_PATH`. Les restrictions de permission ont été conservées. Cela permet le lancement indépendant du bac à sable Windows ; aucun mode sans bac à sable n'a été utilisé.
7. Ajout du dossier contenant `relai.exe` au PATH de toutes les conditions.

Le premier pilote Desktop a répondu avec `/v1` après des lectures refusées par la politique d'exécution. Il est conservé dans `pilot.jsonl` et `raw/easy-relai-0`, mais n'entre pas dans les moyennes. D'autres sondes ont isolé un problème de provisionnement du runtime Desktop. Une lecture réelle a ensuite réussi avec la CLI autonome. Le pilote autonome valide (`pilot-valid.jsonl`, run 99) a exécuté quatre commandes mais a lui aussi répondu `/v1` ; il est également exclu des 18 mesures, conformément au protocole.

Un problème similaire concernant le sélecteur du bac à sable Windows et `--ignore-user-config` est documenté dans [openai/codex #42172](https://github.com/openai/codex/issues/42172). Ce signalement a aidé au diagnostic ; les constats ci-dessus proviennent des journaux locaux.

## Vérifications et limites

- 18 lignes valides, trois numéros de répétition distincts dans chacun des six groupes.
- Compteurs de chaque résultat revérifiés contre ses événements bruts ; aucune mesure n'a zéro commande terminée.
- Les 126 dépôts des 18 fixtures ont un état Git propre après la mesure. Le dépôt RELAI lui-même est inchangé, y compris `build.sh` et le barème original.
- `go build ./cmd/relai` réussit. `go test ./...` échoue sous cet environnement Windows : 6 paquets passent et 8 échouent, avec 50 tests signalés en échec. Le paquet `internal/workspace` passe. Le journal complet est fourni dans `go-test-windows.txt`. Les erreurs concernent notamment les chemins de conventions Windows, des exécutables de test indisponibles et des opérations Git ; elles n'ont pas été corrigées dans ce travail de mesure. Ces résultats ne constituent donc pas une validation générale de la compatibilité Windows de RELAI.
- Le cache, les instructions utilisateur et la surcharge du CLI contribuent aux jetons. Les mesures de temps incluent aussi le démarrage ; elles ne sont pas un benchmark de latence contrôlé. Les événements n'indiquent pas de coût monétaire, donc aucun coût n'est estimé.

## Fichiers

- `results-codex.jsonl` : les 18 mesures principales.
- `summary.txt`, `summary.json` : résumé des six groupes.
- `verification.json` : contrôle des compteurs et de l'état des fixtures.
- `raw/` : preuves par mesure, plus les deux pilotes.
- `diagnostics/` : sondes de lecture ayant précédé la série.
- `run-codex.ps1`, `run-suite.ps1` : copies des lanceurs utilisés. Ils supposent la disposition de travail de cette session : scripts sous `work/`, clone sous `work/RELAI`, CLI sous `work/tools/codex-cli`, binaire sous `work/tools/relai.exe`, livrables sous `outputs/`. Les copies livrées doivent être replacées dans cette disposition avant réexécution.