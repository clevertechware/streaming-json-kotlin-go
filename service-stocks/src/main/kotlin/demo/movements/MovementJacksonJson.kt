package demo.movements

import java.io.OutputStream
import tools.jackson.databind.json.JsonMapper

fun writeMovementsWithJackson(rows: Iterator<MovementRow>, out: OutputStream) {
    JsonMapper.builder().build().createGenerator(out).use { gen ->
        gen.writeStartObject()
        gen.writeArrayPropertyStart("results")
        gen.flush()
        var count = 0
        for (row in rows) {
            gen.writeStartObject()
            gen.writeStringProperty("id", row.id)
            gen.writeStringProperty("warehouseId", row.warehouseId)
            gen.writeStringProperty("code", row.code)
            gen.writeStringProperty("label", row.label)
            gen.writeEndObject()
            if (++count % 100 == 0) gen.flush()
        }
        gen.writeEndArray()
        gen.writeEndObject()
    }
}
