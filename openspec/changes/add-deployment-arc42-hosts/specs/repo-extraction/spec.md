## ADDED Requirements

### Requirement: Infraestructura como código
El extractor SHALL leer Dockerfile, Docker Compose, Kubernetes, OpenShift y plantillas Helm sin
herramientas externas, y SHALL registrar nodos (imágenes, servicios, cargas de trabajo, servicios de
red, ingress/rutas, configuración), sus relaciones y el módulo de código que construyen o ejecutan.
SHALL registrar solo los nombres de las variables de entorno, nunca sus valores.

#### Scenario: Cadena de Kubernetes
- **WHEN** un Ingress enruta a un Service cuyo selector coincide con las etiquetas de un Deployment que usa un Secret
- **THEN** existen las relaciones Ingress→Service (`routes`), Service→Deployment (`selects`) y Deployment→Secret (`config`)

#### Scenario: Referencia entre servicios Compose
- **WHEN** la variable `SPRING_DATASOURCE_URL` de un servicio apunta a `jdbc:oracle:thin:@oracle:1521/X` y existe el servicio `oracle`
- **THEN** existe la relación `env-ref` hacia `oracle` y el valor de la variable no aparece en ninguna salida

### Requirement: Hosts cifrados
Los nombres de host SHALL registrarse solo si hay una clave configurada y SHALL aparecer en las
salidas únicamente como pseudónimos. Su valor real SHALL guardarse cifrado en `hosts.vault.json`.

#### Scenario: Sin clave
- **WHEN** se analiza sin clave
- **THEN** ninguna salida contiene nombres de host ni pseudónimos

#### Scenario: Con clave
- **WHEN** se analiza con `--key-file` y después se ejecuta `traverse reveal` con la misma clave
- **THEN** los informes solo contienen tokens `host-…` y `traverse reveal` devuelve cada token con su host

#### Scenario: Clave incorrecta
- **WHEN** se ejecuta `traverse reveal` con otra clave
- **THEN** la orden falla sin revelar nada
