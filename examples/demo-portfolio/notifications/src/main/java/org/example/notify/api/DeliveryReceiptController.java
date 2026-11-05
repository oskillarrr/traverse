package org.example.notify.api;
import org.springframework.web.bind.annotation.*;
import java.io.ByteArrayInputStream;
import java.security.cert.CertificateFactory;
import java.security.cert.X509Certificate;
import java.util.Base64;
/** Delivery receipts pushed by the messaging partner. Its API gateway terminates mutual TLS
 *  and forwards the partner's certificate in a header. */
@RestController
@RequestMapping("/api/receipts")
public class DeliveryReceiptController {
    @PostMapping("/delivery")
    public void delivered(@RequestHeader("X-Client-Cert") String pem, @RequestBody String receipt) throws Exception {
        X509Certificate partner = (X509Certificate) CertificateFactory.getInstance("X.509")
            .generateCertificate(new ByteArrayInputStream(Base64.getDecoder().decode(pem)));
    }
}
