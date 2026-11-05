package scan

import (
	"sort"
	"strings"
)

// Architectural layers (stable identifiers; reports translate them).
const (
	LPresentation = "Presentation"
	LBusiness     = "Business"
	LPersistence  = "Persistence"
	LIntegration  = "Integration"
	LModel        = "Model"
	LConfig       = "Configuration"
	LCross        = "Cross-cutting"
	LUnknown      = "Unknown"
)

var LayerOrder = []string{LPresentation, LBusiness, LIntegration, LPersistence, LModel, LConfig, LCross, LUnknown}

var stereotypeLayer = map[string]string{
	"Path": LPresentation, "RestController": LPresentation, "Controller": LPresentation, "WebServlet": LPresentation,
	"WebFilter": LPresentation, "ManagedBean": LPresentation, "ViewScoped": LPresentation, "ControllerAdvice": LPresentation,
	"ServerEndpoint": LPresentation, "WebService": LPresentation, "Provider": LPresentation,
	"Remote": LBusiness, "Local": LBusiness, "Stateless": LBusiness, "Stateful": LBusiness, "Singleton": LBusiness, "Service": LBusiness, "MessageDriven": LBusiness,
	"Repository": LPersistence, "Mapper": LPersistence,
	"Entity": LModel, "Embeddable": LModel, "MappedSuperclass": LModel, "XmlRootElement": LModel, "XmlType": LModel,
	"Configuration": LConfig, "SpringBootApplication": LConfig, "ApplicationPath": LConfig, "WebListener": LConfig,
	"FeignClient": LIntegration, "RegisterRestClient": LIntegration, "WebServiceClient": LIntegration,
}

var extendsLayer = map[string]string{
	"HttpServlet": LPresentation, "GenericServlet": LPresentation, "Action": LPresentation, "DispatchAction": LPresentation,
	"LookupDispatchAction": LPresentation, "MappingDispatchAction": LPresentation, "ActionSupport": LPresentation,
	"ActionForm": LPresentation, "ValidatorForm": LPresentation, "DynaActionForm": LPresentation,
	"SessionBean": LBusiness, "MessageDrivenBean": LBusiness, "EntityBean": LPersistence,
	"JpaRepository": LPersistence, "CrudRepository": LPersistence, "PagingAndSortingRepository": LPersistence,
	"HibernateDaoSupport": LPersistence, "JdbcDaoSupport": LPersistence, "SqlMapClientDaoSupport": LPersistence,
	"Application": LConfig, "Exception": LCross, "RuntimeException": LCross,
	// .NET
	"ControllerBase": LPresentation, "Controller": LPresentation, "ApiController": LPresentation, "PageModel": LPresentation,
	"Hub": LPresentation, "ComponentBase": LPresentation, "Page": LPresentation, "UserControl": LPresentation,
	"DbContext": LPersistence, "IdentityDbContext": LPersistence, "ObjectContext": LPersistence,
	"BackgroundService": LBusiness, "IHostedService": LBusiness, "Profile": LCross,
}

var defaultTokens = map[string][]string{
	LPresentation: {"web", "rest", "api", "controller", "controllers", "controlador", "controladores", "servlet", "servlets",
		"action", "actions", "accion", "acciones", "view", "views", "vista", "vistas", "ui", "presentation", "presentacion",
		"resource", "resources", "recurso", "recursos", "endpoint", "endpoints", "jsf", "managedbean", "managedbeans",
		"backing", "form", "forms", "formulario", "formularios", "struts", "mvc", "frontend", "page", "pages", "pagina", "websocket", "graphql",
		"routes", "route", "router", "routers", "handlers", "handler", "components", "component", "templates", "screens", "serializers", "blueprints", "http"},
	LBusiness: {"service", "services", "servicio", "servicios", "business", "negocio", "logic", "logica", "manager", "managers",
		"gestor", "gestores", "facade", "facades", "fachada", "fachadas", "ejb", "ejbs", "usecase", "usecases", "application",
		"process", "proceso", "procesos", "workflow", "rules", "reglas", "batch", "job", "jobs", "scheduler", "tramitacion",
		"specifications", "features", "commands", "queries", "handlers_", "domain_services"},
	LPersistence: {"dao", "daos", "repository", "repositories", "repositorio", "repositorios", "persistence", "persistencia",
		"jdbc", "jpa", "hibernate", "db", "database", "bd", "bbdd", "datos", "data", "store", "mapper", "mappers", "mybatis", "ibatis", "sql", "orm",
		"migrations", "migration", "infrastructure", "infraestructura", "seeders", "dal", "repo", "repos"},
	LIntegration: {"client", "clients", "cliente", "clientes", "integration", "integracion", "integraciones", "connector", "connectors",
		"conector", "conectores", "adapter", "adapters", "adaptador", "adaptadores", "gateway", "proxy", "proxies", "jms", "mq",
		"messaging", "mensajeria", "soap", "ws", "wsclient", "external", "externo", "externos", "remote", "stub", "stubs",
		"kafka", "ftp", "sftp", "mail", "email", "correo", "ldap", "interfaz", "interop"},
	LModel: {"model", "models", "modelo", "modelos", "entity", "entities", "entidad", "entidades", "domain", "dominio", "dto", "dtos",
		"vo", "vos", "to", "bean", "beans", "pojo", "pojos", "bo", "types", "tipos", "enums", "jaxb", "schema", "xsd",
		"interfaces", "contracts", "viewmodels", "valueobjects", "aggregates", "dtos"},
	LConfig: {"config", "configuration", "configuracion", "conf", "cfg", "setup", "bootstrap", "listener", "listeners",
		"startup", "init", "inicializacion", "spring", "boot"},
	LCross: {"util", "utils", "utility", "utilities", "utilidades", "utiles", "common", "commons", "comun", "comunes", "shared",
		"compartido", "helper", "helpers", "exception", "exceptions", "excepcion", "excepciones", "constants", "constantes",
		"log", "logging", "audit", "auditoria", "security", "seguridad", "i18n", "cache", "validation", "validacion", "validator",
		"validators", "converter", "converters", "conversor", "filter", "filters", "filtro", "filtros", "interceptor",
		"interceptors", "aop", "aspect", "aspects", "tools", "herramientas", "support", "framework", "base", "core", "lib",
		"middleware", "middlewares", "helpers", "extensions", "guards", "pipes", "decorators", "hooks",
		"authorization", "authentication", "auth", "guardclauses", "attributes", "logging", "telemetry"},
}

var suffixLayer = []struct{ suf, layer string }{
	{"Controller", LPresentation}, {"Servlet", LPresentation}, {"Handler", LPresentation}, {"Page", LPresentation}, {"Hub", LPresentation}, {"Action", LPresentation}, {"Resource", LPresentation},
	{"Endpoint", LPresentation}, {"Form", LPresentation}, {"Rest", LPresentation}, {"Ws", LPresentation},
	{"Facade", LBusiness}, {"ServiceImpl", LBusiness}, {"UseCase", LBusiness}, {"Command", LBusiness}, {"Query", LBusiness}, {"Service", LBusiness}, {"Manager", LBusiness}, {"BO", LBusiness},
	{"DbContext", LPersistence}, {"DAOImpl", LPersistence}, {"DaoImpl", LPersistence}, {"DAO", LPersistence}, {"Dao", LPersistence}, {"Repository", LPersistence}, {"Mapper", LPersistence},
	{"Client", LIntegration}, {"Proxy", LIntegration}, {"Gateway", LIntegration}, {"Adapter", LIntegration}, {"Connector", LIntegration}, {"Stub", LIntegration},
	{"DTO", LModel}, {"Dto", LModel}, {"VO", LModel}, {"Vo", LModel}, {"TO", LModel}, {"Entity", LModel},
	{"Config", LConfig}, {"Configuration", LConfig}, {"Listener", LConfig}, {"Initializer", LConfig},
	{"Util", LCross}, {"Utils", LCross}, {"Helper", LCross}, {"Exception", LCross}, {"Constants", LCross}, {"Constantes", LCross},
}

var prefixLayer = []struct{ pre, layer string }{
	{"Gestor", LBusiness}, {"Servicio", LBusiness}, {"Dao", LPersistence}, {"Cliente", LIntegration}, {"Util", LCross}, {"Controlador", LPresentation},
}

// Classifier assigns a layer to a type and explains why.
type Classifier struct {
	tokens    map[string]string
	overrides []struct{ prefix, layer string }
}

func NewClassifier(cfg *Config) *Classifier {
	c := &Classifier{tokens: map[string]string{}}
	for layer, toks := range defaultTokens {
		for _, t := range toks {
			c.tokens[t] = layer
		}
	}
	if cfg != nil {
		for layer, toks := range cfg.Layers.Tokens {
			for _, t := range toks {
				c.tokens[strings.ToLower(t)] = layer
			}
		}
		for p, l := range cfg.Layers.Overrides {
			c.overrides = append(c.overrides, struct{ prefix, layer string }{p, l})
		}
		sort.Slice(c.overrides, func(i, j int) bool { return len(c.overrides[i].prefix) > len(c.overrides[j].prefix) })
	}
	return c
}

func (c *Classifier) Classify(pkg, name string, stereotypes, extends []string) (string, string) {
	fqn := name
	if pkg != "" {
		fqn = pkg + "." + name
	}
	for _, o := range c.overrides {
		if fqn == o.prefix || strings.HasPrefix(fqn, o.prefix+".") {
			return o.layer, "config:" + o.prefix
		}
	}
	for _, s := range stereotypes {
		if l, ok := stereotypeLayer[s]; ok {
			return l, "stereotype:@" + s
		}
	}
	for _, e := range extends {
		if i := strings.LastIndexByte(e, '.'); i >= 0 {
			e = e[i+1:]
		}
		if l, ok := extendsLayer[e]; ok {
			return l, "extends:" + e
		}
	}
	segs := strings.FieldsFunc(strings.ToLower(pkg), func(r rune) bool { return r == '.' || r == '_' })
	for i := len(segs) - 1; i >= 0; i-- {
		if l, ok := c.tokens[segs[i]]; ok {
			return l, "package:" + segs[i]
		}
	}
	for _, s := range suffixLayer {
		if strings.HasSuffix(name, s.suf) && len(name) > len(s.suf) {
			return s.layer, "name:*" + s.suf
		}
	}
	for _, p := range prefixLayer {
		if strings.HasPrefix(name, p.pre) && len(name) > len(p.pre) {
			return p.layer, "name:" + p.pre + "*"
		}
	}
	return LUnknown, "none"
}

// forbidden layer dependencies (from -> to).
var forbidden = map[string]map[string]bool{
	LPersistence:  {LPresentation: true},
	LBusiness:     {LPresentation: true},
	LIntegration:  {LPresentation: true},
	LModel:        {LPresentation: true, LBusiness: true, LPersistence: true, LIntegration: true},
	LCross:        {LPresentation: true, LBusiness: true, LPersistence: true},
	LPresentation: {LPersistence: true},
}

func Forbidden(from, to string) bool { return forbidden[from][to] }
