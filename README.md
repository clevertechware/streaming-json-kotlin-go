# Exemple : streamer des mouvements en JSON de Kotlin vers Go

`service-stocks` (Kotlin) écrit `{"results":[…]}` au fil de l'eau depuis la base `movements`.
`service-tasks` (Go) lit ce flux objet par objet et l'insère dans la base `tasks`.

`compose.yaml` crée les bases `movements` (50 000 mouvements pour l'entrepôt `WH-1`) et `tasks`.

## Prérequis

- Docker avec Compose
- Pour le mode par défaut : JDK 25 et Go 1.27
- `psql`, ou `docker compose exec postgres psql`

## Mode par défaut : PostgreSQL dans Docker, services lancés à la main

```bash
docker compose up -d --wait
```

Dans un premier terminal :

```bash
cd service-stocks
./gradlew bootRun
```

Dans un second terminal, une fois `service-stocks` démarré :

```bash
cd service-tasks
go run ./cmd/generate -warehouse WH-1 -task-plan TP-1
```

Sortie attendue : `50000 movements inserted into TP-1 in …`.

## Mode standalone : toute la stack dans Docker

```bash
docker compose --profile standalone up --build --exit-code-from service-tasks
```

`service-tasks` attend que `service-stocks` réponde, lance la génération pour `WH-1` dans le plan `TP-1`, puis
s'arrête. `--exit-code-from` arrête alors la stack et reprend le code de sortie de `service-tasks`.
Sans cette option, `service-stocks` et PostgreSQL restent démarrés.

## Vérification

```bash
psql postgres://demo:demo@localhost:5432/tasks \
  -c "SELECT count(*) FROM task_plan_movements WHERE task_plan_id = 'TP-1'"
```

Résultat attendu : `50000`. Sans `psql` local :

```bash
docker compose exec postgres psql -U demo tasks \
  -c "SELECT count(*) FROM task_plan_movements WHERE task_plan_id = 'TP-1'"
```

## Tests

- `service-tasks` : `cd service-tasks && go test ./...` (Docker requis pour les tests Testcontainers).
- `service-stocks` : voir `service-stocks/README.md`.

## Arrêt

```bash
docker compose --profile standalone down -v
```
