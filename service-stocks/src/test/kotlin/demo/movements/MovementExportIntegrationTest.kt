package demo.movements

import java.io.OutputStream
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.boot.test.web.server.LocalServerPort
import org.springframework.test.context.DynamicPropertyRegistry
import org.springframework.test.context.DynamicPropertySource
import org.springframework.transaction.support.TransactionSynchronizationManager
import org.testcontainers.junit.jupiter.Container
import org.testcontainers.junit.jupiter.Testcontainers
import org.testcontainers.postgresql.PostgreSQLContainer
import tools.jackson.databind.JsonNode
import tools.jackson.databind.json.JsonMapper

private const val SEEDED_ROWS = 1_234

@Testcontainers
@SpringBootTest(
    webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT,
    properties = ["spring.sql.init.mode=always", "spring.sql.init.schema-locations=classpath:seed-movements.sql"],
)
class MovementExportIntegrationTest {

    @LocalServerPort
    private var port = 0

    @Autowired
    private lateinit var exporter: MovementExporter

    private val mapper = JsonMapper.builder().build()

    @Test
    fun `export writes every chunk inside an active read-only transaction`() {
        val out = TransactionStateRecordingStream()

        exporter.export("WH-1", out)

        assertThat(out.writeCount).isPositive()
        assertThat(out.writesOutsideReadOnlyTransaction).isZero()
    }

    @Test
    fun `export returns a valid JSON document with every seeded row of the warehouse in id order`() {
        val response = export("WH-1")

        assertThat(response.statusCode()).isEqualTo(200)
        assertThat(response.headers().firstValue("Content-Type")).hasValueSatisfying { it.startsWith("application/json") }
        val results = results(response)
        assertThat(results).hasSize(SEEDED_ROWS)
        val ids: List<String> = results.map { it.get("id").asString() }
        assertThat(ids).isEqualTo(ids.sorted())
    }

    @Test
    fun `export restores labels containing quotes backslashes and control characters`() {
        val results = results(export("WH-1"))

        val labels: List<String> = results.map { it.get("label").asString() }
        assertThat(labels).contains("Palette \"urgente\"", "Quai C:\\nord", "Ligne 1\nLigne 2")
    }

    @Test
    fun `export leaves out movements of other warehouses`() {
        val results = results(export("WH-1"))

        val warehouseIds: List<String> = results.map { it.get("warehouseId").asString() }
        assertThat(warehouseIds).containsOnly("WH-1")
    }

    @Test
    fun `export returns an empty results array for an unknown warehouse`() {
        val response = export("WH-UNKNOWN")

        assertThat(response.statusCode()).isEqualTo(200)
        assertThat(results(response)).isEmpty()
    }

    @Test
    fun `export rejects a blank warehouse id with 400`() {
        assertThat(export(" ").statusCode()).isEqualTo(400)
    }

    private class TransactionStateRecordingStream : OutputStream() {
        var writeCount = 0
        var writesOutsideReadOnlyTransaction = 0

        override fun write(b: Int) = record()

        override fun write(b: ByteArray, off: Int, len: Int) = record()

        private fun record() {
            writeCount++
            val readOnlyTransaction = TransactionSynchronizationManager.isActualTransactionActive() &&
                TransactionSynchronizationManager.isCurrentTransactionReadOnly()
            if (!readOnlyTransaction) writesOutsideReadOnlyTransaction++
        }
    }

    private fun results(response: HttpResponse<String>): List<JsonNode> =
        mapper.readTree(response.body()).get("results").asSequence().toList()

    private fun export(warehouseId: String): HttpResponse<String> {
        val request = HttpRequest.newBuilder(URI("http://localhost:$port/movements/export"))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString("""{"warehouseId":"$warehouseId"}"""))
            .build()
        return HttpClient.newHttpClient().send(request, HttpResponse.BodyHandlers.ofString())
    }

    companion object {
        @Container
        @JvmStatic
        val postgres = PostgreSQLContainer("postgres:18")

        @JvmStatic
        @DynamicPropertySource
        fun datasource(registry: DynamicPropertyRegistry) {
            registry.add("spring.datasource.url", postgres::getJdbcUrl)
            registry.add("spring.datasource.username", postgres::getUsername)
            registry.add("spring.datasource.password", postgres::getPassword)
        }
    }
}
