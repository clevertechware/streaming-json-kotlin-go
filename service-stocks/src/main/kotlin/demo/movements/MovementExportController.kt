package demo.movements

import org.springframework.http.HttpStatus
import org.springframework.http.MediaType
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.server.ResponseStatusException
import org.springframework.web.servlet.mvc.method.annotation.StreamingResponseBody

data class ExportRequest(val warehouseId: String)

@RestController
class MovementExportController(private val exporter: MovementExporter) {

    @PostMapping("/movements/export", produces = [MediaType.APPLICATION_JSON_VALUE])
    fun export(@RequestBody request: ExportRequest): ResponseEntity<StreamingResponseBody> {
        if (request.warehouseId.isBlank()) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "warehouseId must not be blank")
        }
        return ResponseEntity.ok()
            .contentType(MediaType.APPLICATION_JSON)
            .body(StreamingResponseBody { out -> exporter.export(request.warehouseId, out) })
    }
}
