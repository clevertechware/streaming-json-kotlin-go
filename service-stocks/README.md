# service-stocks (Kotlin) : export JSON en flux

Service Spring MVC qui répond à `POST /movements/export` par un document `{"results":[…]}` écrit au fil de l'eau :
curseur pgjdbc (transaction en lecture seule, `fetch-size` 1 000), tampon de 64 Ko, `flush` toutes les 100 lignes.
Le service Go qui le lit est dans `../service-tasks`.

## Prérequis

- JDK 25 (le wrapper Gradle est fourni ; la toolchain Gradle télécharge un JDK 25 s'il n'est pas installé)
- Docker, pour les tests d'intégration (Testcontainers) et pour `compose.yaml`

## Test

```bash
./gradlew test
```

Les tests unitaires couvrent l'échappement JSON et le rythme des `flush`. Le test d'intégration démarre un
PostgreSQL 17 dans Testcontainers, charge 1 234 mouvements (plus que le `fetch-size` de 1 000) et vérifie que la
réponse est un JSON valide contenant toutes les lignes de l'entrepôt, dans l'ordre de `id`.

## Transaction en lecture seule

Le curseur pgjdbc exige une transaction. Elle doit couvrir toute l'écriture de la réponse, sur le thread asynchrone
qui exécute le `StreamingResponseBody`. Deux écritures sont possibles.

Avec `TransactionTemplate`, la portée de la transaction est visible dans le code :

```kotlin
@Service
class MovementExporter(
    private val jdbc: JdbcTemplate,
    transactionManager: PlatformTransactionManager,
) {
    private val readOnly = TransactionTemplate(transactionManager).apply { isReadOnly = true }

    fun export(warehouseId: String, out: OutputStream) {
        readOnly.executeWithoutResult {
            jdbc.queryForStream(SELECT_BY_WAREHOUSE, rowMapper, warehouseId)
                .use { rows -> writeMovements(rows.iterator(), out) }
        }
    }
}
```

Avec `@Transactional(readOnly = true)`, la version retenue dans `MovementExporter` :

```kotlin
@Service
class MovementExporter(private val jdbc: JdbcTemplate) {

    @Transactional(readOnly = true)
    fun export(warehouseId: String, out: OutputStream) {
        jdbc.queryForStream(SELECT_BY_WAREHOUSE, rowMapper, warehouseId)
            .use { rows -> writeMovements(rows.iterator(), out) }
    }
}
```

L'annotation passe par le proxy Spring : `export` doit être appelée depuis un autre bean (ici le contrôleur) et
rester publique. Le plugin `kotlin("plugin.spring")` ouvre la classe pour le proxy. Si l'appel contourne le proxy,
la transaction disparaît sans erreur et le pilote charge tout le résultat. Le test
`export writes every chunk inside an active read-only transaction` échoue dans ce cas.

## Lancement

Les deux modes de lancement (services à la main ou stack complète dans Docker) sont décrits dans le
`README.md` à la racine du dépôt. Pour ce service seul :

```bash
docker compose up -d --wait   # depuis la racine du dépôt : PostgreSQL seul
./gradlew bootRun
curl -X POST localhost:8080/movements/export \
  -H 'Content-Type: application/json' -d '{"warehouseId":"WH-1"}' -o movements.json
```

La connexion se règle avec `MOVEMENTS_DB_URL`, `MOVEMENTS_DB_USER` et `MOVEMENTS_DB_PASSWORD`.

## Réglages

| Réglage | Valeur | Emplacement |
|---|---|---|
| Lignes lues par le curseur | 1 000 | `spring.jdbc.template.fetch-size` |
| Tampon d'écriture | 64 Ko | `writeMovements` |
| Flush | toutes les 100 lignes | `writeMovements` |
| Délai de la requête asynchrone | 5 min | `spring.mvc.async.request-timeout` |
| Pool d'exports | 4 threads, file de 8 | `AsyncConfig` |
