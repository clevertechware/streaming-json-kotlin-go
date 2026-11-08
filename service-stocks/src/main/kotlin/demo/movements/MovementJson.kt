package demo.movements

import java.io.OutputStream

data class MovementRow(val id: String, val warehouseId: String, val code: String, val label: String)

fun writeMovements(rows: Iterator<MovementRow>, out: OutputStream) {
    val writer = out.writer(Charsets.UTF_8).buffered(64 * 1024)
    writer.append("{\"results\":[")
    writer.flush()
    var count = 0
    for (row in rows) {
        if (count > 0) writer.append(',')
        writer.append(row.toJson())
        if (++count % 100 == 0) writer.flush()
    }
    writer.append("]}")
    writer.flush()
}

fun MovementRow.toJson(): String = buildString {
    append("{\"id\":").appendJsonString(id)
    append(",\"warehouseId\":").appendJsonString(warehouseId)
    append(",\"code\":").appendJsonString(code)
    append(",\"label\":").appendJsonString(label)
    append('}')
}

fun StringBuilder.appendJsonString(value: String): StringBuilder {
    append('"')
    for (char in value) {
        when {
            char == '"' -> append("\\\"")
            char == '\\' -> append("\\\\")
            char == '\n' -> append("\\n")
            char == '\r' -> append("\\r")
            char == '\t' -> append("\\t")
            char == '\b' -> append("\\b")
            char == '\u000C' -> append("\\f")
            char < ' ' -> append("\\u%04x".format(char.code))
            else -> append(char)
        }
    }
    return append('"')
}
