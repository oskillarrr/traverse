package org.example.cases.web;
import javax.servlet.http.HttpServlet;
import org.example.cases.dao.ExpedienteDao;
/* Bad practice kept for the demo: presentation talks to the DAO directly */
public class LegacyExportServlet extends HttpServlet {
    private ExpedienteDao dao = new ExpedienteDao();
}
