## ADDED Requirements

### Requirement: Relaciones de despliegue
El agregador SHALL crear una relación `deploys` cuando un repositorio despliega una imagen cuyo nombre
coincide con un módulo o sistema de otro repositorio del portfolio.

#### Scenario: Repositorio GitOps
- **WHEN** `platform-gitops` declara un Deployment con la imagen `registry/tramitacion/case-web:1.4.2` y `case-management` tiene el módulo `case-web`
- **THEN** existe la relación `platform-gitops → case-management` de tipo `deploys`
