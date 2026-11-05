package report

var labels = map[string]map[string]string{
	"en": {
		"portfolio": "Architecture portfolio", "system": "System", "systems": "Systems", "generated": "Generated",
		"overview": "Overview", "context": "System context", "architecture": "Layered architecture", "modules": "Modules & build",
		"deployables": "Deployment units", "contracts": "Interfaces & integrations", "data": "Data access", "technologies": "Technologies",
		"risks": "Risks & attention points", "components": "Components", "metrics": "Package coupling", "cycles": "Dependency cycles",
		"layerIssues": "Layer violations", "languages": "Languages", "diagnostics": "Diagnostics", "files": "Files",
		"code": "Lines of code", "comment": "Comment lines", "blank": "Blank", "kind": "Kind", "direction": "Direction", "key": "Key",
		"component": "Component", "detail": "Detail", "confidence": "Confidence", "location": "Location", "severity": "Severity",
		"rule": "Rule", "title": "Title", "count": "Count", "message": "Message", "layer": "Layer", "reason": "Why", "package": "Package",
		"types": "Types", "module": "Module", "build": "Build", "packaging": "Packaging", "runtime": "Runtime", "path": "Path",
		"dependencies": "Dependencies", "version": "Version", "scope": "Scope", "resolved": "Resolved", "contextRoot": "Context root",
		"appPaths": "Application paths", "includes": "Includes", "table": "Table", "access": "Access", "filter": "Filter…",
		"links": "Relationships", "from": "From", "to": "To", "sharedTables": "Tables shared by several systems",
		"libraryVersions": "Shared libraries and version divergence", "external": "Dependencies outside the portfolio",
		"techMatrix": "Technology matrix", "landscape": "System landscape", "report": "Report", "exposes": "Exposed", "calls": "Consumed",
		"tables": "Tables", "highRisks": "High risks", "medRisks": "Medium risks", "riskSummary": "Risk summary", "group": "Group",
		"scopeNote": "Static, heuristic analysis of source code. Relationships are evidence-based candidates, not runtime facts: validate before using them in official architecture documents.",
		"noData":    "Nothing detected.", "showing": "Showing first %d of %d rows; the CSV contains all of them.",
		"notes": "Notes", "exports": "Exports", "layers": "Layers", "violations": "violations", "member": "Members", "level": "Level",
		"inbound": "Inbound", "outbound": "Outbound", "dataStores": "Data", "platform": "Platform", "lines": "Lines",
		"diagramNote": "Packages with the most types (max %d). Solid lines: code dependencies; red: forbidden by layering rules. Common prefix: %s",
		"cellsNote":   "Cells show the number of files with evidence of each technology.", "lowConfidence": "low confidence (candidate to validate)",
		"deployment": "Deployment view", "deployImages": "Container images", "deployCompose": "Compose services", "deployIngress": "Ingress / routes",
		"deployServices": "Network services", "deployWorkloads": "Workloads", "deployConfig": "Configuration", "name": "Name", "images": "Images",
		"ports": "Ports", "replicas": "Replicas / schedule", "envKeys": "Environment variables",
		"envKeysNote": "Only variable names are recorded: values are never copied. Registry and ingress host names appear as pseudonyms when a key is configured.",
		"hostsSealed": "Host names are encrypted: the report shows pseudonyms; authorised people decrypt them with `traverse reveal` and the key.",
		"hostsOff":    "Host names are not recorded (no encryption key configured).",
		"files_n":     "files", "systemsCount": "Systems", "linksCount": "Relationships", "sharedCount": "Shared tables", "vcs": "VCS",
		"revision": "Revision", "languagesTop": "Main languages", "deployablesShort": "Deploys", "riskScore": "Risks (H/M/L)",
		"Presentation": "Presentation", "Business": "Business", "Persistence": "Persistence", "Integration": "Integration",
		"Model": "Model", "Configuration": "Configuration", "Cross-cutting": "Cross-cutting", "Unknown": "Unclassified",
		"high": "High", "medium": "Medium", "low": "Low", "info": "Info",
		"critical": "Critical", "unknown": "Unknown", "vulns": "Known vulnerabilities", "vulnSource": "Source: OSV.dev",
		"vulnAsOf": "data as of", "vulnChecked": "library versions checked", "fixed": "Fixed in", "summary": "Summary",
		"malicious": "malicious package", "noVulns": "No known vulnerability in the evaluated versions (this does not mean the libraries are safe: see the unevaluated list and the data date).",
		"unevaluated": "Libraries whose version could not be evaluated", "why_missing": "no version declared",
		"why_unresolved": "unresolved property", "why_external-bom": "managed by a parent POM or BOM outside the repository", "why_range": "version range", "why_ecosystem-not-loaded": "ecosystem not in the database",
		"vulnNote":    "Offline match of declared and committed library versions against the OSV database given with --vulndb. Confidence: high = exact coordinates; medium/low = identified by JAR file name or assembly version. Transitive dependencies are evaluated only when the repository has a lockfile (no build is executed).",
		"affected":    "Affected systems",
		"execSummary": "Executive summary", "mdExport": "Markdown export",
		"transport": "Transport", "authCol": "Auth", "secAuth": "Authentication & transport", "mechanism": "Mechanism", "category": "Category", "mtlsDetected": "mutual TLS / client certificate detected", "weakSpots": "Weak spots", "noAuth": "No authentication or transport-security mechanism detected (verify at the infrastructure layer).", "thirdParties": "Connected applications (third parties)", "extScope": "Scope", "counterpart": "Counterpart", "protocol": "Protocol", "endpoint": "Endpoint",
		"inboundMTLS": "Inbound mTLS", "environments": "Environments", "areas": "Views by area", "areaSystems": "systems of the area", "globalPortfolio": "Global portfolio", "areaNote": "Systems of other areas appear only when they are related to this one.", "relationships": "Relationships with other applications", "callsTo": "What it calls", "calledBy": "Who calls it", "application": "Application", "noCalls": "No outbound call found in the code or the repository configuration.", "noCallers": "No analyzed system calls it.", "interfacesN": "Interfaces", "sharesWith": "Shares data with", "noConsumer": "No identified consumer", "screens": "screens", "others": "other", "consumerNote": "Who consumes an interface is only known when the consumer is analyzed too.", "registeredCerts": "Certificates of trusted applications or servers (registered clients or mTLS peers)", "certificate": "Certificate", "issuer": "Issuer", "validUntil": "Valid until", "status": "Status", "ownIdentity": "Own identity (certificates with private key)", "trustedCAs": "Trusted certification authorities", "repoStructure": "Repository structure", "copiesLeftOut": "Copies left out", "copy": "Copy", "mainLine": "Main line", "overlap": "Overlap", "use": "Use", "userLogin": "User login", "authKind": "Kind", "noUserAuth": "not visible in the code (possibly delegated to the infrastructure: gateway, SSO or application server).",
	},
	"es": {
		"portfolio": "Portfolio de arquitectura", "system": "Sistema", "systems": "Sistemas", "generated": "Generado",
		"overview": "Resumen", "context": "Contexto del sistema", "architecture": "Arquitectura por capas", "modules": "Módulos y build",
		"deployables": "Unidades de despliegue", "contracts": "Interfaces e integraciones", "data": "Acceso a datos", "technologies": "Tecnologías",
		"risks": "Riesgos y puntos de atención", "components": "Componentes", "metrics": "Acoplamiento de paquetes", "cycles": "Ciclos de dependencias",
		"layerIssues": "Violaciones de capas", "languages": "Lenguajes", "diagnostics": "Diagnósticos", "files": "Ficheros",
		"code": "Líneas de código", "comment": "Líneas de comentario", "blank": "En blanco", "kind": "Tipo", "direction": "Dirección", "key": "Clave",
		"component": "Componente", "detail": "Detalle", "confidence": "Confianza", "location": "Ubicación", "severity": "Severidad",
		"rule": "Regla", "title": "Título", "count": "Nº", "message": "Mensaje", "layer": "Capa", "reason": "Motivo", "package": "Paquete",
		"types": "Tipos", "module": "Módulo", "build": "Build", "packaging": "Empaquetado", "runtime": "Runtime", "path": "Ruta",
		"dependencies": "Dependencias", "version": "Versión", "scope": "Ámbito", "resolved": "Resuelta", "contextRoot": "Context root",
		"appPaths": "Rutas de aplicación", "includes": "Incluye", "table": "Tabla", "access": "Acceso", "filter": "Filtrar…",
		"links": "Relaciones", "from": "Origen", "to": "Destino", "sharedTables": "Tablas compartidas por varios sistemas",
		"libraryVersions": "Librerías compartidas y divergencia de versiones", "external": "Dependencias fuera del portfolio",
		"techMatrix": "Matriz de tecnologías", "landscape": "Mapa de sistemas", "report": "Informe", "exposes": "Expuestos", "calls": "Consumidos",
		"tables": "Tablas", "highRisks": "Riesgos altos", "medRisks": "Riesgos medios", "riskSummary": "Resumen de riesgos", "group": "Grupo",
		"scopeNote": "Análisis estático y heurístico del código fuente. Las relaciones son candidatas basadas en evidencias, no hechos de ejecución: valídalas antes de usarlas en documentación oficial de arquitectura.",
		"noData":    "No se ha detectado nada.", "showing": "Se muestran las primeras %d de %d filas; el CSV las contiene todas.",
		"notes": "Notas", "exports": "Exportaciones", "layers": "Capas", "violations": "violaciones", "member": "Miembros", "level": "Nivel",
		"inbound": "Entrada", "outbound": "Salida", "dataStores": "Datos", "platform": "Plataforma", "lines": "Líneas",
		"diagramNote": "Paquetes con más tipos (máx. %d). Líneas: dependencias de código; en rojo, prohibidas por las reglas de capas. Prefijo común: %s",
		"cellsNote":   "Cada celda indica el número de ficheros con evidencias de la tecnología.", "lowConfidence": "confianza baja (candidata a validar)",
		"deployment": "Vista de despliegue", "deployImages": "Imágenes de contenedor", "deployCompose": "Servicios Compose", "deployIngress": "Ingress / rutas",
		"deployServices": "Servicios de red", "deployWorkloads": "Cargas de trabajo", "deployConfig": "Configuración", "name": "Nombre", "images": "Imágenes",
		"ports": "Puertos", "replicas": "Réplicas / planificación", "envKeys": "Variables de entorno",
		"envKeysNote": "Solo se registran los nombres de las variables: nunca sus valores. Los hosts de registro e ingress aparecen como pseudónimos cuando hay clave configurada.",
		"hostsSealed": "Los nombres de host están cifrados: el informe muestra pseudónimos y las personas autorizadas los descifran con `traverse reveal` y la clave.",
		"hostsOff":    "No se registran nombres de host (no hay clave de cifrado configurada).",
		"files_n":     "ficheros", "systemsCount": "Sistemas", "linksCount": "Relaciones", "sharedCount": "Tablas compartidas", "vcs": "VCS",
		"revision": "Revisión", "languagesTop": "Lenguajes principales", "deployablesShort": "Despliega", "riskScore": "Riesgos (A/M/B)",
		"Presentation": "Presentación", "Business": "Negocio", "Persistence": "Persistencia", "Integration": "Integración",
		"Model": "Modelo", "Configuration": "Configuración", "Cross-cutting": "Transversal", "Unknown": "Sin clasificar",
		"high": "Alta", "medium": "Media", "low": "Baja", "info": "Info",
		"critical": "Crítica", "unknown": "Desconocida", "vulns": "Vulnerabilidades conocidas", "vulnSource": "Fuente: OSV.dev",
		"vulnAsOf": "datos a fecha", "vulnChecked": "versiones de librerías evaluadas", "fixed": "Corregida en", "summary": "Resumen",
		"malicious": "paquete malicioso", "noVulns": "Ninguna vulnerabilidad conocida en las versiones evaluadas (no significa que las librerías sean seguras: revisa la lista de no evaluables y la fecha de los datos).",
		"unevaluated": "Librerías cuya versión no se ha podido evaluar", "why_missing": "sin versión declarada",
		"why_unresolved": "propiedad sin resolver", "why_external-bom": "la gestiona un POM padre o BOM fuera del repositorio", "why_range": "rango de versiones", "why_ecosystem-not-loaded": "ecosistema no incluido en la base",
		"vulnNote":    "Cruce sin conexión de las versiones de librerías declaradas o incluidas en el repositorio con la base OSV indicada en --vulndb. Confianza: alta = coordenadas exactas; media/baja = identificada por el nombre del JAR o la versión del ensamblado. Las dependencias transitivas solo se evalúan si el repositorio tiene lockfile (no se ejecuta ningún build).",
		"affected":    "Sistemas afectados",
		"execSummary": "Resumen ejecutivo", "mdExport": "Exportación Markdown",
		"transport": "Transporte", "authCol": "Auth", "secAuth": "Autenticación y transporte", "mechanism": "Mecanismo", "category": "Categoría", "mtlsDetected": "TLS mutuo / certificado de cliente detectado", "weakSpots": "Puntos débiles", "noAuth": "No se ha detectado ningún mecanismo de autenticación o seguridad de transporte (verifícalo en la capa de infraestructura).", "thirdParties": "Aplicaciones conectadas (terceros)", "extScope": "Ámbito", "counterpart": "Contraparte", "protocol": "Protocolo", "endpoint": "Endpoint",
		"inboundMTLS": "mTLS entrante", "environments": "Entornos", "areas": "Vistas por área", "areaSystems": "sistemas del área", "globalPortfolio": "Portfolio global", "areaNote": "Los sistemas de otras áreas solo aparecen si se relacionan con esta.", "relationships": "Relaciones con otras aplicaciones", "callsTo": "A quién llama", "calledBy": "Quién le llama", "application": "Aplicación", "noCalls": "No se ha encontrado ninguna llamada saliente en el código ni en la configuración del repositorio.", "noCallers": "Ningún sistema analizado le llama.", "interfacesN": "Interfaces", "sharesWith": "Comparte datos con", "noConsumer": "Sin consumidor identificado", "screens": "pantallas", "others": "otras", "consumerNote": "Quién consume una interfaz solo se sabe si el consumidor también se analiza.", "registeredCerts": "Certificados de aplicaciones o servidores de confianza (clientes registrados o pares mTLS)", "certificate": "Certificado", "issuer": "Emisor", "validUntil": "Válido hasta", "status": "Estado", "ownIdentity": "Identidad propia (certificados con clave privada)", "trustedCAs": "Autoridades de certificación de confianza", "repoStructure": "Estructura del repositorio", "copiesLeftOut": "Copias omitidas", "copy": "Copia", "mainLine": "Línea principal", "overlap": "Solape", "use": "Uso", "userLogin": "Login de usuarios", "authKind": "Tipo", "noUserAuth": "no visible en el código (posiblemente delegado en la infraestructura: pasarela, SSO o servidor de aplicaciones).",
		"exposes_d": "expone", "calls_d": "invoca", "publishes_d": "publica", "subscribes_d": "consume", "uses_d": "usa",
		"defines_d": "define", "read_d": "lee", "write_d": "escribe", "excludes_d": "excluye",
	},
}

func tr(lang, key string) string {
	if v, ok := labels[lang][key]; ok {
		return v
	}
	if v, ok := labels["en"][key]; ok {
		return v
	}
	return key
}

func trDir(lang, d string) string {
	if v, ok := labels[lang][d+"_d"]; ok {
		return v
	}
	return d
}
