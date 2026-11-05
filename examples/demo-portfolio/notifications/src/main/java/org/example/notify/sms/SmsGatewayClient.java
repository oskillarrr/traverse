package org.example.notify.sms;
import org.springframework.web.client.RestTemplate;
/** Sends SMS through an external provider (demo: plaintext HTTP on purpose). */
public class SmsGatewayClient {
    private final RestTemplate rest = new RestTemplate();
    public void send(String phone, String text) {
        rest.postForObject("http://sms-gateway.proveedor.example.net/api/v1/sms", text, String.class);
    }
}
