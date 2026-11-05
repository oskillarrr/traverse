package org.example.legacy.gestion.web;
import javax.ws.rs.*;
/**
 * This class exposes expedientes.
 */
@Path("/expedientes")
@Produces("application/json")
public class ExpedienteRest {
    @GET
    @Path("/{id}")
    public String uno(@PathParam("id") String id) { return id; }

    public String ayudante() { return ""; }

    @POST
    @Path(
      "/alta")
    public void alta() { }
}
