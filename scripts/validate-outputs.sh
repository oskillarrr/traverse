#!/usr/bin/env sh
# Validates an Traverse output folder against the official tools and schemas.
# Traverse itself never downloads anything; fetch these once into $TOOLS (see docs/standards.md):
#   structurizr-cli (unzipped), plantuml.jar, schemas: sarif-schema-2.1.0.json,
#   bom-1.6.schema.json (+ spdx.schema.json, jsf-0.82.schema.json), archimate3_*.xsd (+ xml.xsd)
# Requires: bash, java, xmllint, python3 with jsonschema; optional: mmdc (Mermaid CLI).
# Usage: TOOLS=/path/to/tools scripts/validate-outputs.sh OUTPUT_DIR
set -u
OUT="${1:?output dir}"; T="${TOOLS:?set TOOLS}"; rc=0
echo "== Structurizr DSL"
for f in "$OUT"/workspace.dsl "$OUT"/systems/*/workspace.dsl; do
  msg=$(bash "$T/structurizr/structurizr.sh" validate -workspace "$f" 2>&1 | grep -v JAVA_TOOL_OPTIONS)
  [ -n "$msg" ] && { echo "FAIL $f: $msg"; rc=1; }
done
echo "== C4-PlantUML"
java -jar "$T/plantuml.jar" -checkonly "$OUT/landscape.puml" || rc=1
echo "== ArchiMate 3.1 exchange format"
xmllint --noout --schema "$T/schemas/archimate3_Diagram.xsd" "$OUT/architecture.archimate.xml" || rc=1
echo "== SARIF 2.1.0 and CycloneDX 1.6"
python3 -I - "$OUT" "$T/schemas" <<'PY' || rc=1
import json, glob, sys, jsonschema
from referencing import Registry, Resource
out, sc = sys.argv[1], sys.argv[2]
sarif = json.load(open(f"{sc}/sarif-schema-2.1.0.json"))
cdx = json.load(open(f"{sc}/bom-1.6.schema.json"))
reg = Registry()
for n in ["spdx.schema.json", "jsf-0.82.schema.json", "bom-1.6.schema.json"]:
    s = json.load(open(f"{sc}/{n}"))
    reg = reg.with_resource(n, Resource.from_contents(s)).with_resource(s.get("$id", n), Resource.from_contents(s))
bad = 0
for f in [f"{out}/findings.sarif"] + glob.glob(f"{out}/systems/*/findings.sarif"):
    for e in jsonschema.Draft7Validator(sarif).iter_errors(json.load(open(f))):
        bad += 1; print("SARIF", f, e.message[:200])
V = jsonschema.validators.validator_for(cdx)
for f in glob.glob(f"{out}/systems/*/sbom.cdx.json"):
    for e in V(cdx, registry=reg).iter_errors(json.load(open(f))):
        bad += 1; print("CycloneDX", f, e.message[:200])
print("JSON schemas:", "OK" if bad == 0 else f"{bad} errors")
sys.exit(1 if bad else 0)
PY
if command -v mmdc >/dev/null; then
  echo "== Mermaid"
  for f in "$OUT"/landscape.mmd "$OUT"/systems/*/architecture.mmd; do
    mmdc -q -i "$f" -o /tmp/traverse-mmd.svg >/dev/null 2>&1 || { echo "FAIL $f"; rc=1; }
  done
  echo "== Mermaid blocks in Markdown (arc42)"
  for f in "$OUT"/portfolio.md "$OUT"/systems/*/arc42.md; do
    mmdc -q -i "$f" -o /tmp/traverse-md-out.md >/dev/null 2>&1 || { echo "FAIL $f"; rc=1; }
  done
fi
[ $rc -eq 0 ] && echo "ALL OUTPUTS VALID" || echo "VALIDATION FAILED"
exit $rc
