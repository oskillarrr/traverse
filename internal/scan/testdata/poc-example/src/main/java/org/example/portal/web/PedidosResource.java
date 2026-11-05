package org.example.portal.web;
import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import org.example.portal.service.PedidosService;
@Path("/pedidos")
public class PedidosResource {
    private PedidosService service;
    @GET
    @Path("/estado")
    public String estado() { return "OK"; }
}
