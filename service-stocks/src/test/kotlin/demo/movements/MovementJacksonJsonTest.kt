package demo.movements

import java.io.ByteArrayOutputStream
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class MovementJacksonJsonTest {
    @Test
    fun `jackson generator writes the same document as the hand written json`() {
        val rows = listOf(
            MovementRow("MV-1", "WH-1", "C-1", "Palette \"A\" \\ ligne\n2"),
            MovementRow("MV-2", "WH-1", "C-2", "Carton"),
        )
        val byHand = ByteArrayOutputStream().also { writeMovements(rows.iterator(), it) }.toString(Charsets.UTF_8)
        val byJackson = ByteArrayOutputStream().also { writeMovementsWithJackson(rows.iterator(), it) }.toString(Charsets.UTF_8)
        assertEquals(byHand, byJackson)
    }
}
