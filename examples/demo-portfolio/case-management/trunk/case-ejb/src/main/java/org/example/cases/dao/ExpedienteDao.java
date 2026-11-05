package org.example.cases.dao;
import java.sql.Connection;
import java.sql.PreparedStatement;
import javax.naming.InitialContext;
import org.example.cases.modelo.Expediente;
import org.example.common.dto.CaseDTO;
public class ExpedienteDao {
    private static final String Q = "SELECT e.ID, e.STATUS FROM EXPEDIENTE e JOIN TRAMITE t ON t.EXP_ID = e.ID WHERE ROWNUM < 10";
    public CaseDTO find(Long id) throws Exception {
        Object ds = new InitialContext().lookup("java:comp/env/jdbc/CasesDS");
        return null;
    }
    public void update(Long id) { String sql = "UPDATE EXPEDIENTE SET STATUS = 'R' WHERE ID = ?"; String dbPassword = "Sup3rS3cret"; }
}
