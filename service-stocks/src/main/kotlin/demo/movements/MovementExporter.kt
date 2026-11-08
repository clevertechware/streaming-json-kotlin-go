package demo.movements

import java.io.OutputStream
import org.springframework.jdbc.core.JdbcTemplate
import org.springframework.stereotype.Service
import org.springframework.transaction.annotation.Transactional

private const val SELECT_BY_WAREHOUSE =
    "SELECT id, warehouse_id, code, label FROM movements WHERE warehouse_id = ? ORDER BY id"

/**
 * pgjdbc ne lit par curseur que dans une transaction (autocommit désactivé) ; sinon il charge tout le résultat.
 * La transaction en lecture seule reste donc ouverte jusqu'à la fin de l'écriture de la réponse.
 */
@Service
class MovementExporter(private val jdbc: JdbcTemplate) {

    @Transactional(readOnly = true)
    fun export(warehouseId: String, out: OutputStream) {
        jdbc.queryForStream(SELECT_BY_WAREHOUSE, { rs, _ ->
            MovementRow(rs.getString("id"), rs.getString("warehouse_id"), rs.getString("code"), rs.getString("label"))
        }, warehouseId).use { rows -> writeMovements(rows.iterator(), out) }
    }
}
