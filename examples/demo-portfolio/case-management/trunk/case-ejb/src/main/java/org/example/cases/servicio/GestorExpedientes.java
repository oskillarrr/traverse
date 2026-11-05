package org.example.cases.servicio;
import javax.ejb.Stateless;
import javax.annotation.Resource;
import javax.jms.Queue;
import org.apache.log4j.Logger;
import org.example.cases.dao.ExpedienteDao;
import org.example.cases.integracion.NotificacionesClient;
import org.example.common.dto.CaseDTO;
@Stateless
public class GestorExpedientes {
    private static final Logger LOG = Logger.getLogger(GestorExpedientes.class);
    @Resource(mappedName = "java:/jms/queue/CaseResolved")
    private Queue resolvedQueue;
    private ExpedienteDao dao;
    private NotificacionesClient notif;
    public CaseDTO buscar(Long id) { return dao.find(id); }
    public void resolver(Long id) { dao.update(id); notif.enviar(id); }
}
