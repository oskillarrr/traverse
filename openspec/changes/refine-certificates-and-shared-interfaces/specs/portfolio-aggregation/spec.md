## ADDED Requirements

### Requirement: Interfaz implementada por varios sistemas
Cuando la interfaz llamada por un consumidor la exponen varios sistemas analizados, cada relación
SHALL tener como máximo confianza media, SHALL registrar el número de candidatos y SHALL indicarlo
en los informes.

#### Scenario: Interfaz de aviso implementada por dos aplicaciones
- **WHEN** un sistema llama a `CallbackWSService` y dos sistemas exponen `callbackWS`
- **THEN** hay dos relaciones de confianza media con `candidates = 2`
