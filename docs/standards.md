# Standards and how outputs are validated

| Output | Standard | Reference |
|---|---|---|
| `workspace.dsl` | C4 model, Structurizr DSL | https://c4model.com · https://docs.structurizr.com/dsl/language |
| `landscape.puml` | C4-PlantUML (PlantUML stdlib) | https://plantuml.com/stdlib |
| `architecture.archimate.xml` | ArchiMate® 3.1/3.2 Model Exchange File Format, The Open Group | https://www.opengroup.org/xsd/archimate/ |
| `findings.sarif` | SARIF 2.1.0, OASIS Standard | https://docs.oasis-open.org/sarif/sarif/v2.1.0/ |
| `sbom.cdx.json` | CycloneDX 1.6 (ECMA-424) with Package URL; `vulnerabilities` (VDR) when `--vulndb` is used | https://cyclonedx.org/specification/overview/ · https://github.com/package-url/purl-spec |
| `--vulndb` input | OSV schema (OpenSSF) · CVSS v3.1 base score (FIRST) | https://ossf.github.io/osv-schema/ · https://www.first.org/cvss/v3.1/specification-document |
| `*.mmd` | Mermaid | https://mermaid.js.org |

## Validating an output folder

Traverse never downloads anything. Fetch the validators once, on a machine allowed to, into a folder `$TOOLS`:

```
$TOOLS/structurizr/            structurizr-cli (unzipped release from github.com/structurizr/cli)
$TOOLS/plantuml.jar            release from github.com/plantuml/plantuml
$TOOLS/schemas/sarif-schema-2.1.0.json        github.com/oasis-tcs/sarif-spec (sarif-2.1/schema)
$TOOLS/schemas/bom-1.6.schema.json, spdx.schema.json, jsf-0.82.schema.json   github.com/CycloneDX/specification (schema/)
$TOOLS/schemas/archimate3_Model.xsd, archimate3_View.xsd, archimate3_Diagram.xsd, xml.xsd
                               opengroup.org/xsd/archimate/3.1/ (point Model.xsd's xml.xsd import to the local copy)
```

Then:

```sh
TOOLS=/path/to/tools scripts/validate-outputs.sh reports/portfolio
```

Requirements: bash, Java 17+, `xmllint`, Python 3 with `jsonschema`; optional Mermaid CLI.

## Model mapping

| Traverse fact | C4 (Structurizr) | ArchiMate 3.1 |
|---|---|---|
| System (repository) | Software System | Application Component |
| Build module / deployable | Container | Application Component (C4) · Artifact + Realization (ArchiMate) |
| Layer within a module | Component | — |
| Exposed contracts by kind | Relationship descriptions | Application Interface (Composition) |
| Tables, datasets | External system "DB-TABLE"/"DATASET" | Data Object + Access (read/write) |
| Runtime, app server, DBMS | Technology on containers | System Software + Serving |
| Call (HTTP, SOAP, EJB, CICS, program) | Relationship | Serving (provider → consumer) |
| Messaging, shared dataset | Relationship | Flow |
| Shared table | Relationship | Association |
| Manifest group | `group` | Grouping + Aggregation |
