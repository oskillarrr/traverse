package org.example.portal.service;
import javax.ejb.Stateless;
import javax.naming.InitialContext;
import org.example.portal.dao.PedidosDao;
import org.example.portal.integracion.ProveedorSoap;
@Stateless
public class PedidosService {
    private PedidosDao dao;
    private ProveedorSoap integracion;
    public void procesar() { }
}
