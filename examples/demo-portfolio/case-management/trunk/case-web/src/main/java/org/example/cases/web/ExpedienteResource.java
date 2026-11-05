package org.example.cases.web;

import javax.ws.rs.*;
import javax.ejb.EJB;
import org.example.cases.servicio.GestorExpedientes;
import org.example.common.dto.CaseDTO;

/**
 * Gestión de expedientes. This class exposes the REST API.
 */
@Path("/expedientes")
@Produces("application/json")
public class ExpedienteResource {
    @EJB
    private GestorExpedientes gestor;

    @GET
    @Path("/{id}")
    public CaseDTO get(@PathParam("id") Long id) { return gestor.buscar(id); }

    @POST
    @Path("/{id}/resolver")
    public void resolve(@PathParam("id") Long id) { gestor.resolver(id); }
}
