package demo.movements

import java.io.ByteArrayOutputStream
import java.io.OutputStream
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.params.ParameterizedTest
import org.junit.jupiter.params.provider.CsvSource
import org.junit.jupiter.api.Test

class MovementJsonTest {

    @ParameterizedTest
    @CsvSource(
        delimiter = '|',
        value = [
            "plain|\"plain\"",
            "say \"hi\"|\"say \\\"hi\\\"\"",
            "C:\\quai|\"C:\\\\quai\"",
            "été Réception|\"été Réception\"",
        ],
    )
    fun `appendJsonString escapes quotes and backslashes and keeps accents`(raw: String, expected: String) {
        assertThat(StringBuilder().appendJsonString(raw).toString()).isEqualTo(expected)
    }

    @Test
    fun `appendJsonString escapes control characters`() {
        val json = StringBuilder().appendJsonString("a\nb\rc\td\u0008e\u000Cf\u0001g").toString()

        assertThat(json).isEqualTo("\"a\\nb\\rc\\td\\be\\ff\\u0001g\"")
    }

    @Test
    fun `toJson writes the four fields in a fixed order`() {
        val json = MovementRow("MV-1", "WH-1", "C-1", "Palette").toJson()

        assertThat(json).isEqualTo("""{"id":"MV-1","warehouseId":"WH-1","code":"C-1","label":"Palette"}""")
    }

    @Test
    fun `writeMovements writes an empty results array when there is no row`() {
        val out = ByteArrayOutputStream()

        writeMovements(emptyList<MovementRow>().iterator(), out)

        assertThat(out.toString(Charsets.UTF_8)).isEqualTo("""{"results":[]}""")
    }

    @Test
    fun `writeMovements separates objects with commas and no newline`() {
        val rows = listOf(MovementRow("1", "S", "c1", "l1"), MovementRow("2", "S", "c2", "l2"))
        val out = ByteArrayOutputStream()

        writeMovements(rows.iterator(), out)

        assertThat(out.toString(Charsets.UTF_8)).isEqualTo(
            """{"results":[{"id":"1","warehouseId":"S","code":"c1","label":"l1"},{"id":"2","warehouseId":"S","code":"c2","label":"l2"}]}""",
        )
    }

    @Test
    fun `writeMovements flushes the opening, every 100 rows and the end`() {
        val flushes = FlushCountingStream()
        val rows = (1..250).map { MovementRow("$it", "S", "c", "l") }

        writeMovements(rows.iterator(), flushes)

        assertThat(flushes.count).isEqualTo(1 + 2 + 1)
    }

    private class FlushCountingStream : OutputStream() {
        var count = 0
        override fun write(b: Int) = Unit
        override fun flush() {
            count++
        }
    }
}
