## ADDED Requirements

### Requirement: ArchiMate 3.1 Exchange Format
El sistema SHALL exportar el portfolio en ArchiMate 3.1 Model Exchange File Format con sistemas,
interfaces, objetos de datos, software de sistema, artefactos, agrupaciones, relaciones y una
vista de portfolio, válido contra el XSD oficial.

#### Scenario: Importación en una herramienta EA
- **WHEN** se valida `architecture.archimate.xml` con `archimate3_Diagram.xsd`
- **THEN** la validación es correcta

### Requirement: SARIF 2.1.0
El sistema SHALL exportar los hallazgos de riesgo (severidad distinta de `info`) en SARIF 2.1.0
con huellas estables y URI relativos.

#### Scenario: Plataforma de code scanning
- **WHEN** se valida `findings.sarif` con el JSON Schema oficial
- **THEN** la validación es correcta y cada resultado incluye `partialFingerprints.traverseFindingId/v1`

### Requirement: SBOM CycloneDX 1.6
El sistema SHALL exportar por sistema un SBOM CycloneDX 1.6 con PURL para los ecosistemas
soportados, excluyendo `dependencyManagement`, y sin versión cuando la declarada es un rango.

#### Scenario: Rango npm
- **WHEN** una dependencia npm declara `^4.18.2`
- **THEN** el componente no tiene `version` y conserva `traverse:declaredVersion`

### Requirement: Validación reproducible de formatos
El proyecto SHALL incluir un script que valide todas las salidas con las herramientas y esquemas
oficiales sin que Traverse descargue nada.

#### Scenario: Ejecución del script
- **WHEN** se ejecuta `scripts/validate-outputs.sh` sobre una carpeta de salida
- **THEN** informa `ALL OUTPUTS VALID` o los errores de cada fichero
