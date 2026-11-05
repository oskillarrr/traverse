package org.example.cases.integracion;
import java.security.KeyStore;
import javax.net.ssl.KeyManagerFactory;
import javax.net.ssl.SSLContext;
import org.apache.commons.httpclient.HttpClient;
import org.apache.commons.httpclient.methods.PostMethod;
public class NotificacionesClient {
    private static final String URL = "https://notify.internal.example:8443/notifications/api/messages/send";
    public void enviar(Long id) throws Exception {
        // client certificate (mutual TLS) towards the notifications service
        KeyManagerFactory kmf = KeyManagerFactory.getInstance("SunX509");
        kmf.init(KeyStore.getInstance("PKCS12"), new char[0]);
        SSLContext ctx = SSLContext.getInstance("TLSv1.2");
        ctx.init(kmf.getKeyManagers(), null, null);
        HttpClient c = new HttpClient();
    }
}
