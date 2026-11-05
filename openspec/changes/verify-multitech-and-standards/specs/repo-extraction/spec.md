## ADDED Requirements

### Requirement: Análisis estructural multilenguaje
El extractor SHALL obtener componentes, capas, dependencias y endpoints de Java, Kotlin y C#,
y SHALL obtener módulos, imports resueltos, rutas HTTP y tablas ORM de JavaScript/TypeScript,
Python, PHP, Go y Ruby.

#### Scenario: Controlador ASP.NET con token de ruta
- **WHEN** una clase `CatalogController` tiene `[Route("api/[controller]")]` y un método `[HttpGet("{id}")]`
- **THEN** se registra el contrato `GET /api/Catalog/{id}`

#### Scenario: Prefijo de router en el mismo fichero
- **WHEN** un `APIRouter(prefix="/orders")` declara `@router.get("/{order_id}")`
- **THEN** se registra `GET /orders/{order_id}`

### Requirement: Análisis mainframe
El extractor SHALL registrar programas COBOL, copybooks, jobs JCL, procedimientos y mapsets BMS
como componentes. SHALL registrar como dependencias CALL, EXEC CICS LINK/XCTL (resolviendo
literales, `VALUE` y `MOVE` a variables), COPY, mapas y `EXEC PGM=`. SHALL registrar como
contratos transacciones CSD, ficheros CICS (y su dataset), datasets JCL con su disposición y
colas TS/TD.

#### Scenario: Transferencia vía variable
- **WHEN** `MOVE 'COADM01C' TO WS-PGM` precede a `EXEC CICS XCTL PROGRAM(WS-PGM)`
- **THEN** existe la dependencia hacia `COADM01C`

#### Scenario: Formato fijo
- **WHEN** las columnas 73-80 contienen números de secuencia
- **THEN** no se interpretan como nombres de programa

### Requirement: URL solo con cliente HTTP
Una URL literal SHALL registrarse como llamada saliente solo si el fichero usa un cliente HTTP.

#### Scenario: Enlace a vídeo en una migración
- **WHEN** una migración de Django contiene `https://www.youtube.com/watch?v=...`
- **THEN** no se registra ninguna llamada HTTP
