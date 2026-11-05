## ADDED Requirements

### Requirement: Enlace con context root versionado
Una llamada HTTP SHALL enlazar con un sistema cuyo context root deducido lleve la versión del artefacto
(`/sign-gateway-2.3`) cuando la ruta llamada coincide sin esa versión.

#### Scenario: Cliente de una pasarela de firma
- **WHEN** un sistema llama a `/sign-gateway/signService` y la pasarela `sign-gw` expone `/sign-gateway-2.3/signService`
- **THEN** el portfolio registra la relación del cliente con `sign-gw`

### Requirement: Vistas por área
Cuando el manifiesto define más de un grupo, el sistema SHALL generar una vista por grupo a partir del
portfolio global. Cada vista SHALL incluir los sistemas del grupo y los de otros grupos relacionados con
él, con sus relaciones.

#### Scenario: Área con relación externa
- **WHEN** `portal` (grupo Canales) llama a `sign-gw` (grupo Integración)
- **THEN** la vista de Integración muestra `sign-gw`, `portal` y la relación entre ellos, y no muestra sistemas sin relación con el área
