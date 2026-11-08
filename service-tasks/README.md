# service-tasks (Go)

Lit le flux `POST /movements/export` de `service-stocks` (Kotlin) avec `json.Decoder` (`Token`, `More`, `Decode`)
et insère les mouvements par lots de 100 dans la table `task_plan_movements` de la base `tasks`.
La table est créée par `../postgres/init/03-tasks.sql`, que les tests d'intégration réutilisent.

## Prérequis

- Go 1.27
- Docker (tests d'intégration Testcontainers ; PostgreSQL via `../compose.yaml`)

## Build, lancement, tests

```bash
go build -o ./bin/service-tasks main.go
go vet ./...
go test ./...
go run main.go -warehouse WH-1 -task-plan TP-1
```

`go test` vérifie avec [goleak](https://github.com/uber-go/goleak) qu'aucune goroutine ne reste en vie à la fin des tests
du paquet `internal/export`.

| Variable | Défaut |
|---|---|
| `SERVICE_STOCKS_URL` | `http://localhost:8080` |
| `TASKS_DATABASE_URL` | `postgres://demo:demo@localhost:5432/tasks` |

Chaque génération supprime d'abord les lignes existantes du plan de tâches, puis insère le flux.
Un flux tronqué termine la commande en erreur (code 1).

## Ce que l'exemple montre

- `internal/export/decode.go` : `skipToResults`, `decodeMovement`, `readInBatches` (lots de 100).
- `internal/export/pipeline.go` : canal de capacité 4, goroutine lectrice et goroutine d'insertion
  dans un `errgroup` ; la première erreur annule l'autre goroutine.
- `internal/export/client.go` : client sans `Client.Timeout`, `ResponseHeaderTimeout` de 90 s, statut vérifié
  avant la lecture du corps.
- `internal/export/store.go` : `INSERT` multi-lignes paramétré.
