## ADDED Requirements

### Requirement: Certificados públicos sin datos personales
El sistema SHALL leer los certificados públicos del repositorio sin descifrar nada:
- ficheros PEM y DER, también sin extensión;
- entradas de confianza de truststores JKS/JCEKS;
- la cadena en claro de las entradas con clave.

De cada certificado SHALL registrar sujeto, emisor, fecha de caducidad, si es CA y si es la identidad
propia. Para certificados de persona física SHALL NOT registrar el sujeto ni el alias. Los CN con forma
de nombre de host SHALL cifrarse como el resto de hosts.

#### Scenario: Aplicación registrada en una pasarela
- **WHEN** `registeredApps/` contiene un certificado DER sin extensión con `CN=ORDERS APP`
- **THEN** se registra como certificado de confianza con ese sujeto, su emisor y su caducidad

#### Scenario: Certificado de persona
- **WHEN** el sujeto incluye un número de identificación personal o atributos de nombre y apellidos
- **THEN** se registra como certificado personal sin sujeto ni alias

#### Scenario: Certificado caducado
- **WHEN** un certificado que no es de CA caducó antes de la fecha del análisis
- **THEN** se genera un hallazgo `SEC-CERT-EXPIRED` con su fecha de caducidad
