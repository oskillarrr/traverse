"""Authoring helper: regenerates internal/rules/default.json from readable Python raw strings.
Contributors may edit default.json directly; this script is only a convenience."""
import json, sys
JVM=["Java","Kotlin","Groovy","Scala","JSP"]
JAVA=["Java","JSP"]
CONF=["XML","Properties","YAML","TOML/INI","Maven POM","Gradle","Ant","Ivy","JSF/Facelets"]
JS=["JavaScript","TypeScript","Vue","HTML","JSP","JSF/Facelets","ASP.NET","Classic ASP","PHP","Velocity/Freemarker"]
NET=["C#","VB.NET/VB6","ASP.NET","XML","MSBuild"]
SQLL=["SQL","PL/SQL","Transact-SQL"]
CODE=["Java","Kotlin","Groovy","Scala","JSP","C#","VB.NET/VB6","JavaScript","TypeScript","Python","PHP","Go","Ruby","C","C++","Delphi/Pascal","PowerBuilder"]
R=[]
def r(id,title,cat,tech,pat=None,langs=None,sev="info",conf="medium",match=None,nomatch=None,msg="",tags=None,path=None,exclude=None):
    d={"id":id,"title":title,"category":cat,"technology":tech,"severity":sev,"confidence":conf}
    if langs: d["languages"]=langs
    if pat: d["pattern"]=pat
    if path: d["pathPattern"]=path
    if exclude: d["exclude"]=exclude
    if msg: d["message"]=msg
    if tags: d["tags"]=tags
    ex={}
    if match: ex["match"]=match if isinstance(match,list) else [match]
    if nomatch: ex["noMatch"]=nomatch if isinstance(nomatch,list) else [nomatch]
    if ex: d["examples"]=ex
    R.append(d)

# ---------- Java / Jakarta EE / frameworks ----------
r("JAVA-JAXRS","JAX-RS REST API","Web","JAX-RS",r"\b(javax|jakarta)\.ws\.rs\b",JVM,match="import javax.ws.rs.GET;",nomatch="// import javax.ws.rs.GET;")
r("JAVA-SPRING-MVC","Spring MVC / REST controllers","Web","Spring MVC",r"@(RestController|Controller|RequestMapping|GetMapping|PostMapping|PutMapping|DeleteMapping)\b|\borg\.springframework\.web\b",JVM,match="@RestController",nomatch="/* @RestController */")
r("JAVA-SPRING-BOOT","Spring Boot","Framework","Spring Boot",r"@SpringBootApplication\b|\borg\.springframework\.boot\b|spring-boot-starter",JVM+["Maven POM","Gradle"],match="@SpringBootApplication")
r("JAVA-SPRING-CORE","Spring Framework (DI)","Framework","Spring Framework",r"\borg\.springframework\.(beans|context|stereotype)\b|@Autowired\b",JVM,match="@Autowired private Foo foo;")
r("JAVA-SERVLET","Servlet API","Web","Servlet",r"\b(javax|jakarta)\.servlet\b|\bextends\s+HttpServlet\b",JVM,match="public class X extends HttpServlet {")
r("JAVA-JSF","JavaServer Faces","Web","JSF",r"\b(javax|jakarta)\.faces\b|\borg\.primefaces\b|xmlns:h=\"http://(java\.sun\.com|xmlns\.jcp\.org)/jsf",JVM+["JSF/Facelets"],match="import javax.faces.bean.ManagedBean;")
r("JAVA-RICHFACES","RichFaces / ICEfaces / a4j (end of life)","Web","RichFaces/ICEfaces",r"\borg\.(richfaces|ajax4jsf|icefaces)\b|xmlns:(rich|a4j|ice)=",JVM+["JSF/Facelets","XML"],"high",match="import org.richfaces.component.UIDataTable;",msg="Component library discontinued; blocks JSF/Jakarta upgrades.",tags=["eol"])
r("JAVA-STRUTS1","Struts 1 (end of life since 2013)","Web","Struts 1",r"\borg\.apache\.struts\.(action|actions|upload|validator|util|config)\b",JVM,"high","high",match="import org.apache.struts.action.Action;",nomatch="import org.apache.struts2.ServletActionContext;",msg="Unsupported framework with known unpatched vulnerabilities.",tags=["eol","security"])
r("JAVA-STRUTS2","Struts 2","Web","Struts 2",r"\bcom\.opensymphony\.xwork2\b|\borg\.apache\.struts2\b",JVM,"medium",match="import org.apache.struts2.ServletActionContext;",msg="Framework with a history of critical RCE CVEs: verify version.",tags=["security"])
r("JAVA-GWT","Google Web Toolkit","Web","GWT",r"\bcom\.google\.gwt\b",JVM+["XML"],"medium",match="import com.google.gwt.core.client.EntryPoint;",tags=["legacy"])
r("JAVA-DWR","DWR (Direct Web Remoting, end of life)","Web","DWR",r"\borg\.directwebremoting\b|\buk\.ltd\.getahead\.dwr\b",JVM+["XML","JavaScript"],"high",match="org.directwebremoting.servlet.DwrServlet",tags=["eol"])
r("JAVA-EJB","Enterprise JavaBeans","Business","EJB",r"\b(javax|jakarta)\.ejb\b|@(Stateless|Stateful|Singleton|MessageDriven)\b",JVM,match="@Stateless")
r("JAVA-EJB2","EJB 2.x component model","Business","EJB 2.x",r"\bextends\s+(EJBHome|EJBObject|EJBLocalHome|EJBLocalObject)\b|\bimplements\s+(SessionBean|EntityBean|MessageDrivenBean)\b",JVM,"high","high",match="public class XBean implements SessionBean {",msg="EJB 2.x / CMP entity beans are removed from Jakarta EE.",tags=["migration"])
r("JAVA-CDI","CDI / JSR-330 injection","Framework","CDI",r"\b(javax|jakarta)\.inject\b|\b(javax|jakarta)\.enterprise\.(context|inject)\b",JVM,match="import javax.inject.Inject;")
r("JAVA-JMS","JMS messaging","Integration","JMS",r"\b(javax|jakarta)\.jms\b|@JmsListener\b|\bJmsTemplate\b",JVM,match="import javax.jms.Queue;",nomatch="// import javax.jms.Queue;")
r("JAVA-JNDI","JNDI lookups","Integration","JNDI",r"\bnew\s+InitialContext\s*\(|\bjavax\.naming\.(InitialContext|Context)\b|java:(comp|global|jboss|app|module)/",JVM+CONF,match="Context ctx = new InitialContext();")
r("JAVA-JPA","JPA persistence","Persistence","JPA",r"\b(javax|jakarta)\.persistence\b",JVM,match="import javax.persistence.Entity;")
r("JAVA-HIBERNATE","Hibernate native API","Persistence","Hibernate",r"\borg\.hibernate\.(Session|SessionFactory|Criteria|Query|cfg|Transaction)\b",JVM,match="import org.hibernate.Session;")
r("JAVA-HBM-MAPPING","Hibernate XML mappings (hbm.xml)","Persistence","Hibernate",path=r"\.hbm\.xml$",tags=["legacy"])
r("JAVA-JDBC","Plain JDBC","Persistence","JDBC",r"\bjava\.sql\.(Connection|DriverManager|PreparedStatement|Statement|CallableStatement)\b|\bDriverManager\.getConnection\b",JVM,match="import java.sql.PreparedStatement;")
r("JAVA-MYBATIS","MyBatis / iBATIS","Persistence","MyBatis/iBATIS",r"\borg\.apache\.ibatis\b|\bcom\.ibatis\b|\borg\.mybatis\b|<!DOCTYPE\s+(sqlMap|mapper)",JVM+["XML"],match="import org.apache.ibatis.annotations.Select;")
r("JAVA-SPRING-JDBC","Spring JDBC","Persistence","Spring JDBC",r"\b(Named)?JdbcTemplate\b",JVM,match="private JdbcTemplate jdbc;")
r("JAVA-JAXWS","SOAP web services (JAX-WS)","Integration","JAX-WS/SOAP",r"\b(javax|jakarta)\.xml\.ws\b|@WebService\b|@WebMethod\b|\bjavax\.xml\.soap\b",JVM,match="@WebService(serviceName=\"X\")")
r("JAVA-AXIS1","Apache Axis 1 (end of life)","Integration","Axis 1",r"\borg\.apache\.axis\.[a-z]",JVM+["XML"],"high","high",match="import org.apache.axis.client.Call;",nomatch="import org.apache.axis2.client.ServiceClient;",msg="Unmaintained since 2006 with known vulnerabilities.",tags=["eol","security"])
r("JAVA-AXIS2","Apache Axis2","Integration","Axis2",r"\borg\.apache\.axis2\b",JVM+["XML"],"medium",match="import org.apache.axis2.client.ServiceClient;")
r("JAVA-CXF","Apache CXF","Integration","CXF",r"\borg\.apache\.cxf\b|xmlns:jaxws=",JVM+["XML"],match="import org.apache.cxf.frontend.ClientProxy;")
r("JAVA-JAXB","JAXB (javax.xml.bind, removed from JDK 11)","Integration","JAXB",r"\bjavax\.xml\.bind\b",JVM,"medium","high",match="import javax.xml.bind.JAXBContext;",msg="Needs explicit dependency on Java 11+ and jakarta.xml.bind on Jakarta EE 9+.",tags=["migration"])
r("JAVA-HTTP-CLIENT","Outbound HTTP client","Integration","HTTP client",r"\b(RestTemplate|WebClient|HttpURLConnection|CloseableHttpClient|OkHttpClient|HttpClient)\b|@FeignClient\b|@RegisterRestClient\b|\b(javax|jakarta)\.ws\.rs\.client\b",JVM,match="RestTemplate rt = new RestTemplate();",nomatch="/* new URL( */")
r("JAVA-COMMONS-HTTPCLIENT3","Commons HttpClient 3.x (end of life)","Integration","HttpClient 3",r"\borg\.apache\.commons\.httpclient\b",JVM,"medium",match="import org.apache.commons.httpclient.HttpClient;",tags=["eol"])
r("JAVA-KAFKA","Apache Kafka","Integration","Kafka",r"\borg\.apache\.kafka\b|@KafkaListener\b|bootstrap\.servers",JVM+CONF,match="@KafkaListener(topics=\"t\")")
r("JAVA-RABBITMQ","RabbitMQ / AMQP","Integration","RabbitMQ",r"\bcom\.rabbitmq\b|\borg\.springframework\.amqp\b|@RabbitListener\b",JVM,match="import com.rabbitmq.client.Channel;")
r("JAVA-ACTIVEMQ","ActiveMQ / Artemis / HornetQ","Integration","ActiveMQ",r"\borg\.apache\.activemq\b|\borg\.hornetq\b",JVM+CONF,match="import org.apache.activemq.ActiveMQConnectionFactory;")
r("JAVA-IBMMQ","IBM MQ","Integration","IBM MQ",r"\bcom\.ibm\.mq\b|\bcom\.ibm\.msg\.client\b",JVM+CONF,match="import com.ibm.mq.MQQueueManager;")
r("JAVA-TIBCO","TIBCO","Integration","TIBCO",r"\bcom\.tibco\b",JVM+CONF,match="import com.tibco.tibjms.TibjmsConnectionFactory;")
r("JAVA-QUARTZ","Quartz scheduler","Scheduling","Quartz",r"\borg\.quartz\b",JVM+["XML","Properties"],match="import org.quartz.Job;")
r("JAVA-TIMERS","EJB / Spring timers","Scheduling","Timers",r"@Schedule\b|@Scheduled\b|\bTimerService\b|@Timeout\b",JVM,match="@Schedule(hour=\"*\")")
r("JAVA-MAIL","E-mail (JavaMail)","Integration","SMTP",r"\b(javax|jakarta)\.mail\b|\bJavaMailSender\b|mail\.smtp\.",JVM+CONF,match="import javax.mail.Transport;")
r("JAVA-LDAP","LDAP / directory","Integration","LDAP",r"\bjavax\.naming\.(ldap|directory)\b|\bLdapContext\b|\bLdapTemplate\b|ldaps?://",JVM+CONF,match="import javax.naming.ldap.LdapContext;")
r("ANY-FTP","FTP / SFTP transfer","Integration","FTP/SFTP",r"\borg\.apache\.commons\.net\.ftp\b|\bcom\.jcraft\.jsch\b|\bChannelSftp\b|\bs?ftp://|\bFtpWebRequest\b|\bftplib\b",None,match="import com.jcraft.jsch.ChannelSftp;")
r("JAVA-FILESYSTEM","Local filesystem access","Integration","Filesystem",r"\bnew\s+File(InputStream|OutputStream|Reader|Writer)?\s*\(|\bFiles\.(newBufferedReader|newBufferedWriter|readAllBytes|readAllLines|write|copy|move)\s*\(",JVM,"low","low",match="new FileInputStream(f)",msg="Stateful local storage: affects horizontal scaling and containerization.",tags=["cloud"])
r("JAVA-SPRING-SECURITY","Spring Security","Security","Spring Security",r"\borg\.springframework\.security\b|<http\s+auto-config",JVM+["XML"],match="import org.springframework.security.core.Authentication;")
r("JAVA-JAAS","JAAS / container security","Security","JAAS",r"\bjavax\.security\.auth\b|@RolesAllowed\b|@DeclareRoles\b|\bisUserInRole\s*\(",JVM,match="@RolesAllowed(\"admin\")")
r("JAVA-JBOSS-API","JBoss / WildFly proprietary API","Platform","JBoss/WildFly",r"\borg\.jboss\.(?:[a-z])|\borg\.wildfly\.",JVM+["XML"],"medium",match="import org.jboss.logging.Logger;",msg="Couples the application to the JBoss/WildFly runtime.",tags=["vendor-lockin"])
r("JAVA-WEBLOGIC","WebLogic proprietary API","Platform","WebLogic",r"\bweblogic\.[a-z]",JVM+["XML"],"medium",match="import weblogic.jndi.WLInitialContextFactory;",tags=["vendor-lockin"])
r("JAVA-WEBSPHERE","WebSphere proprietary API","Platform","WebSphere",r"\bcom\.ibm\.(websphere|ws)\.",JVM+["XML"],"medium",match="import com.ibm.websphere.cache.DistributedMap;",tags=["vendor-lockin"])
r("JAVA-JAVAX-EE","javax.* Java EE namespace (pre Jakarta EE 9)","Migration","Java EE (javax)",r"\bjavax\.(servlet|ejb|persistence|ws\.rs|jms|faces|inject|enterprise|validation|transaction|annotation\.security|mail|xml\.ws|websocket|json|batch|resource)\b",JVM+["XML"],"medium","high",match="import javax.servlet.http.HttpServletRequest;",nomatch="import javax.swing.JFrame;",msg="Requires the javax→jakarta namespace migration for Jakarta EE 9+ runtimes.",tags=["migration:jakarta"])
r("JAVA-JDK-INTERNAL","Internal JDK APIs (sun.*)","Migration","JDK internals",r"\b(com\.sun|sun)\.(misc|reflect|security|nio\.ch|awt|net\.www|org\.apache)\.",JVM,"high","high",match="import sun.misc.BASE64Encoder;",msg="Strongly encapsulated since Java 16; some removed.",tags=["migration:jdk"])
r("JAVA-CORBA","CORBA (removed from JDK 11)","Migration","CORBA",r"\borg\.omg\.|\bjavax\.rmi\.CORBA\b|\bjavax\.activity\b",JVM,"high","high",match="import org.omg.CORBA.ORB;",tags=["migration:jdk"])
r("JAVA-RMI","Java RMI","Integration","RMI",r"\bjava\.rmi\b|\bextends\s+UnicastRemoteObject\b",JVM,"medium",match="import java.rmi.Remote;")
r("JAVA-APPLET","Applets / Java Web Start (removed)","Migration","Applet/JNLP",r"\bjava\.applet\b|\bextends\s+J?Applet\b|<applet\b|\bjavax\.jnlp\b",JVM+["HTML"],"high","high",match="public class A extends Applet {",tags=["eol"])
r("JAVA-JNLP-FILE","Java Web Start descriptor","Migration","Applet/JNLP",path=r"\.jnlp$",sev="high",tags=["eol"])
r("JAVA-NASHORN","Nashorn JavaScript engine (removed in JDK 15)","Migration","Nashorn",r"getEngineByName\s*\(\s*\"(nashorn|javascript|js|JavaScript|ECMAScript)\"",JVM,"medium","high",match="m.getEngineByName(\"nashorn\")",tags=["migration:jdk"])
r("JAVA-SECURITY-MANAGER","SecurityManager (deprecated for removal)","Migration","SecurityManager",r"\bSystem\.setSecurityManager\b|\bnew\s+SecurityManager\b|\bAccessController\.doPrivileged\b",JVM,"medium",match="System.setSecurityManager(sm);",tags=["migration:jdk"])
r("JAVA-FINALIZE","finalize() (deprecated for removal)","Migration","Finalization",r"\bprotected\s+void\s+finalize\s*\(\s*\)",JVM,"low",match="protected void finalize() {",tags=["migration:jdk"])
r("JAVA-UNMANAGED-THREADS","Unmanaged threads","Runtime","Threads",r"\bnew\s+Thread\s*\(|\bExecutors\.new\w+\s*\(",JVM,"low","low",match="new Thread(r).start();",msg="Unmanaged threads inside a container bypass its lifecycle and context.")
r("JAVA-SYSTEM-EXIT","System.exit in application code","Runtime","System.exit",r"\bSystem\.exit\s*\(",JVM,"low",match="System.exit(1);")
r("JAVA-LOG4J1","Log4j 1.x (end of life, CVEs)","Logging","Log4j 1.x",r"\borg\.apache\.log4j\.(Logger|Category|PropertyConfigurator|LogManager|Level|xml|net|Appender)\b",JVM,"high","high",match="import org.apache.log4j.Logger;",nomatch="import org.apache.logging.log4j.Logger;",msg="End of life since 2015 with unpatched vulnerabilities (e.g. CVE-2019-17571).",tags=["eol","security"])
r("JAVA-LOG4J1-CONFIG","Log4j 1.x configuration file","Logging","Log4j 1.x",path=r"(^|/)log4j\.(properties|xml)$",sev="medium",tags=["eol"])
r("JAVA-LOG4J2","Log4j 2","Logging","Log4j 2",r"\borg\.apache\.logging\.log4j\b",JVM,"low",match="import org.apache.logging.log4j.LogManager;",msg="Verify version >= 2.17.1 (Log4Shell family).",tags=["security"])
r("JAVA-SLF4J","SLF4J / Logback","Logging","SLF4J",r"\borg\.slf4j\b|\bch\.qos\.logback\b",JVM,match="import org.slf4j.Logger;")
r("JAVA-JSP-SCRIPTLET","JSP scriptlets","Web","JSP",r"<%[^@=!\-]",["JSP"],"low",match="<% int x = 1; %>",nomatch="<%@ page import=\"x\" %>",msg="Business logic embedded in views.")
r("JAVA-REPORTING","Reporting / documents (Jasper, iText, POI)","Library","Reporting",r"\bnet\.sf\.jasperreports\b|\bcom\.lowagie\.text\b|\bcom\.itextpdf\b|\borg\.apache\.poi\b",JVM,match="import org.apache.poi.hssf.usermodel.HSSFWorkbook;")
r("JAVA-COMMONS-COLLECTIONS3","Commons Collections 3 (deserialization gadget)","Security","Commons Collections 3",r"\borg\.apache\.commons\.collections\.",JVM,"low",match="import org.apache.commons.collections.CollectionUtils;",nomatch="import org.apache.commons.collections4.CollectionUtils;",tags=["security"])
r("JAVA-DESERIALIZATION","Java native deserialization","Security","Serialization",r"\bnew\s+ObjectInputStream\s*\(|\bcom\.thoughtworks\.xstream\b|\bXMLDecoder\b",JVM,"medium","low",match="new ObjectInputStream(in)",msg="Review for untrusted input (deserialization attacks).",tags=["security"])

# ---------- Cross-language: config, security, data ----------
r("CFG-EXTERNAL","Externalized configuration","Configuration","Configuration",r"\bSystem\.(getenv|getProperty)\s*\(|@Value\s*\(\s*\"\$\{|@ConfigProperty\b|\bprocess\.env\.|\bos\.environ\b|\bEnvironment\.GetEnvironmentVariable\b|\bConfigurationManager\.AppSettings\b",CODE,match="String u = System.getenv(\"URL\");")
r("SEC-SECRET-CONFIG","Possible hard-coded credential in configuration","Security","Secrets",r"\b[\w.\-]*(?i:password|passwd|pwd|secret|api[_.\-]?key|access[_.\-]?token|client[_.\-]?secret)(?:[._\-][\w.\-]*|[A-Z0-9][\w.\-]*)?\s*[:=]\s*[\"']?[^\s\"'${}<>#]{4,}",["Properties","YAML","TOML/INI"],"high","low",match=["db.password=s3cr3t!","app.firmaPwd=s3cr3t!","CLIENT_SECRET: abcd1234","keystore.pwd.alias=s3cr3t!"],nomatch=["db.password=${DB_PASSWORD}","# password=example","role.secretary.label=Secretary","job.secretario.label=Secretario","url.pwd.error=https://example.org/error","password.reset.page=/account/reset"],msg="Value is never copied to the report; check location and rotate if real.",tags=["security"],exclude=r"[:=]\s*[\"']?(?:[a-zA-Z][a-zA-Z0-9+.\-]*://|/)")
r("SEC-SECRET-XML","Possible hard-coded credential in XML","Security","Secrets",r"(?i)<(password|user-password|secret|api-key)>\s*[^<$\s]{4,}\s*</|(?i)\b(password|secret)\s*=\s*\"[^\"$]{4,}\"",["XML"],"high","low",match="<password>s3cr3t!</password>",nomatch="<password>${db.pass}</password>",msg="Value is never copied to the report.",tags=["security"])
r("SEC-SECRET-CODE","Possible hard-coded credential in code","Security","Secrets",r"\b\w*(?i:password|passwd|pwd|secret|apikey|api_key|token|(?:jks|keystore|truststore)[_.]?(?:key|pass))(?:_\w*|[A-Z0-9]\w*)?\s*=\s*\"[^\"$\s]{4,}\"",CODE,"high","low",match=["String dbPassword = \"s3cr3t!\";","private static final String API_KEY = \"abcd1234\";","String secretKey = \"s3cr3t!\";"],nomatch=["String dbPassword = getPwd();","String secretario = \"Juan\";","String tokenizerName = \"standard\";","String tokenUrl = \"https://example.org/oauth/token\";"],msg="Value is never copied to the report.",tags=["security"],exclude=r"=\s*\"(?:[a-zA-Z][a-zA-Z0-9+.\-]*://|/)")
r("SEC-SECRET-URL","Credentials embedded in a connection string or URL","Security","Secrets",r"\b[a-zA-Z][a-zA-Z0-9+.\-]*://[^/\s:@\"'<>{}$]+:[^/\s@\"'<>{}$]{3,}@|\bjdbc:oracle:(?:thin|oci):[^/\s:@\"'<>{}$]+/[^@\s\"'<>{}$]{3,}@",CONF+JVM+["XML"],"high","medium",match=["db.url=jdbc:oracle:thin:APP_USER/s3cr3t@//db.example.org:1521/SVC","amqp.uri=amqp://app:s3cr3t@mq.example.org:5672","spring.datasource.url=postgresql://app:s3cr3t@db.example.org/app"],nomatch=["db.url=jdbc:oracle:thin:@//db.example.org:1521/SVC","url=https://example.org/a@b","uri=amqp://${MQ_USER}:${MQ_PASS}@mq.example.org"],msg="User and password inside the connection string; the value is never copied to the report. Move it to a secret store and rotate it.",tags=["security"])
r("SEC-PRIVATE-KEY","Private key committed","Security","Secrets",r"-----BEGIN (RSA |EC |DSA |OPENSSH |ENCRYPTED )?PRIVATE KEY-----",None,"high","high",match="-----BEGIN RSA PRIVATE KEY-----",tags=["security"])
r("SEC-KEYSTORE-FILE","Keystore / private key file committed","Security","Secrets",path=r"(?i)\.(jks|jceks|bks|p12|pfx|keystore|pem|key|keys)([._-]?(old|bak|backup|orig|copia|copy|\d+))?$",sev="high",msg="A key container in the repository may hold private keys: check its content (truststores with only public certificates are reported as info).",tags=["security"])
r("SEC-CERT-FILE","Public certificate file committed","Security","Certificates",path=r"(?i)\.(cer|crt|der)([._-]?(old|bak|backup|orig|copia|copy|\d+))?$",tags=["pki"])
r("CFG-IP-LITERAL","Hard-coded IP address","Configuration","Infrastructure",r"\b(?:(?:25[0-5]|2[0-4]\d|1?\d?\d)\.){3}(?:25[0-5]|2[0-4]\d|1?\d?\d)\b",["Properties","YAML","TOML/INI","XML"],"low","low",match="server.host=10.1.2.3",msg="Environment-specific address in the repository.",tags=["cloud"])
for vid,vname,pat,ex in [
    ("ORACLE","Oracle Database",r"jdbc:oracle:|oracle\.jdbc\.|\bOracle\d*[ig]?Dialect\b|\bojdbc\d*\b","jdbc:oracle:thin:@db:1521:X"),
    ("DB2","IBM DB2",r"jdbc:db2:|jdbc:as400:|com\.ibm\.db2\.|\bDB2\d*Dialect\b","jdbc:db2://db:50000/X"),
    ("SQLSERVER","SQL Server",r"jdbc:sqlserver:|jdbc:jtds:|\bSQLServer\d*Dialect\b|\bSystem\.Data\.SqlClient\b|Initial Catalog=","jdbc:sqlserver://db;databaseName=X"),
    ("POSTGRES","PostgreSQL",r"jdbc:postgresql:|\bPostgreSQL\d*Dialect\b|\bpsycopg2?\b|\bNpgsql\b","jdbc:postgresql://db/x"),
    ("MYSQL","MySQL / MariaDB",r"jdbc:(mysql|mariadb):|\bMySQL\d*\w*Dialect\b|\bpymysql\b|\bmysqli?_","jdbc:mysql://db/x"),
    ("INFORMIX","Informix",r"jdbc:informix|\bInformixDialect\b","jdbc:informix-sqli://db:1526/x"),
    ("SYBASE","Sybase",r"jdbc:sybase:|\bSybase\w*Dialect\b","jdbc:sybase:Tds:db:5000"),
    ("MONGODB","MongoDB",r"mongodb(\+srv)?://|\bcom\.mongodb\b|\bpymongo\b|\bmongoose\b","mongodb://db/x"),
    ("REDIS","Redis",r"redis://|\bredis\.clients\.jedis\b|\bio\.lettuce\b|\bStackExchange\.Redis\b","redis://cache:6379"),
    ("EMBEDDED","Embedded database",r"jdbc:(h2|hsqldb|derby):|\bsqlite3?\b","jdbc:h2:mem:test"),
]:
    r("DB-"+vid,vname,"Database",vname,pat,None,match=ex)
r("SQL-ORACLE-DIALECT","Oracle-specific SQL","Database","Oracle SQL",r"(?i)\bCONNECT\s+BY\b|\bROWNUM\b|\bNVL2?\s*\(|\bDECODE\s*\(|\bSYSDATE\b|\(\+\)|\bFROM\s+DUAL\b",SQLL+JAVA+["C#","PowerBuilder","PHP","Python"],"low",match="select nvl(a,0) from dual",msg="Proprietary SQL hinders database portability.",tags=["vendor-lockin"])
r("SQL-STORED-PROCEDURES","Stored procedures / packages","Database","PL/SQL",r"(?i)\bCREATE\s+(OR\s+REPLACE\s+)?(PROCEDURE|FUNCTION|PACKAGE|TRIGGER)\b|\{\s*call\s+\w",SQLL+JAVA,match="CREATE OR REPLACE PROCEDURE p AS BEGIN NULL; END;",msg="Business logic in the database.")
r("SQL-DBLINK","Database links","Integration","DB link",r"(?i)\b(from|join|into|update)\s+[\w.$#]+@[A-Za-z_]\w*|\bCREATE\s+(PUBLIC\s+)?DATABASE\s+LINK\b",SQLL,"medium",match="select * from t@remote_db",msg="Hidden database-to-database integration.")
r("SQL-UTL-NETWORK","Database-initiated network/file access","Integration","DB network",r"(?i)\b(UTL_HTTP|UTL_SMTP|UTL_TCP|UTL_FILE|UTL_MAIL|DBMS_AQ|xp_cmdshell|sp_send_dbmail)\b",SQLL,"medium",match="v := UTL_HTTP.REQUEST(url);",msg="The database itself integrates with external systems.")
r("SQL-DB-JOBS","Database jobs","Scheduling","DB jobs",r"(?i)\b(DBMS_JOB|DBMS_SCHEDULER|sp_add_job)\b",SQLL,match="DBMS_SCHEDULER.CREATE_JOB(",msg="Batch processes scheduled inside the database.")

# ---------- Mainframe / legacy ----------
r("COBOL-CICS","CICS transactions","Platform","CICS",r"(?i)\bEXEC\s+CICS\b",["COBOL","PL/I"],match="           EXEC CICS RETURN END-EXEC.")
r("COBOL-SQL","Embedded SQL (DB2)","Persistence","Embedded SQL",r"(?i)\bEXEC\s+SQL\b",["COBOL","PL/I","C","RPG"],match="           EXEC SQL SELECT A INTO :B FROM T END-EXEC.")
r("COBOL-MQ","IBM MQ calls from COBOL","Integration","IBM MQ",r"(?i)\bCALL\s+'MQ(PUT1?|GET|OPEN|CONN|CLOSE)'",["COBOL"],match="           CALL 'MQPUT' USING HCONN.")
r("VB6-PROJECT","Visual Basic 6 project","Platform","VB6",path=r"\.(vbp|frm|frx)$",sev="high",tags=["eol"])
r("DELPHI-PROJECT","Delphi project","Platform","Delphi",path=r"\.(dpr|dproj|dfm)$",sev="medium",tags=["legacy"])

# ---------- .NET ----------
r("NET-WCF","WCF services","Integration","WCF",r"\bSystem\.ServiceModel\b|\[(ServiceContract|OperationContract)\b|<system\.serviceModel>",NET,"medium",match="[ServiceContract]",msg="WCF server is not available on modern .NET (CoreWCF required).",tags=["migration:dotnet"])
r("NET-ASMX","ASMX web services","Integration","ASMX",r"\[WebMethod\b|\bSystem\.Web\.Services\b",NET,"medium",match="[WebMethod]",tags=["migration:dotnet"])
r("NET-WEBFORMS","ASP.NET Web Forms","Web","Web Forms",r"\bSystem\.Web\.UI\b|<%@\s*Page\b|\brunat=\"server\"",NET,"high",match="using System.Web.UI;",msg="Not supported on .NET Core / .NET 5+.",tags=["migration:dotnet"])
r("NET-MVC","ASP.NET MVC / Core","Web","ASP.NET MVC",r"\bSystem\.Web\.Mvc\b|\bMicrosoft\.AspNetCore\b|\[ApiController\]",NET,match="using Microsoft.AspNetCore.Mvc;")
r("NET-REMOTING",".NET Remoting","Integration",".NET Remoting",r"\bSystem\.Runtime\.Remoting\b",NET,"high","high",match="using System.Runtime.Remoting;",tags=["migration:dotnet"])
r("NET-EF","Entity Framework","Persistence","Entity Framework",r"\bSystem\.Data\.Entity\b|\bMicrosoft\.EntityFrameworkCore\b|\bDbContext\b",NET,match="public class Ctx : DbContext {")
r("NET-ADO","ADO.NET","Persistence","ADO.NET",r"\b(SqlConnection|OleDbConnection|OdbcConnection|OracleConnection|SqlCommand)\b",NET,match="var c = new SqlConnection(cs);")
r("NET-DESKTOP","WinForms / WPF desktop","Platform","Desktop .NET",r"\bSystem\.Windows\.Forms\b|\bSystem\.Windows\.(Controls|Markup)\b",NET,match="using System.Windows.Forms;")
r("NET-FRAMEWORK",".NET Framework (Windows only)","Platform",".NET Framework",r"<TargetFrameworkVersion>\s*v[1-4]\.|<TargetFramework>\s*net[1-4]\d",["MSBuild","XML"],"medium","high",match="<TargetFrameworkVersion>v4.5</TargetFrameworkVersion>",tags=["migration:dotnet"])

# ---------- JavaScript / front-end ----------
r("JS-JQUERY","jQuery","Frontend","jQuery",r"\bjQuery\b|\$\(\s*(document|function|window)\b|jquery[\w.\-]*\.js",JS,match="$(document).ready(function(){});")
r("JS-ANGULARJS","AngularJS 1.x (end of life)","Frontend","AngularJS",r"\bangular\.module\s*\(|\bng-app\b",JS,"medium",match="angular.module('app', [])",msg="End of life since January 2022.",tags=["eol"])
r("JS-ANGULAR","Angular","Frontend","Angular",r"@angular/core",JS+["npm"],match="import { Component } from '@angular/core';")
r("JS-REACT","React","Frontend","React",r"from\s+['\"]react['\"]|require\(\s*['\"]react['\"]\s*\)|\"react\"\s*:",JS+["npm"],match="import React from 'react';")
r("JS-VUE","Vue.js","Frontend","Vue",r"from\s+['\"]vue['\"]|\bnew\s+Vue\s*\(|\"vue\"\s*:",JS+["npm"],match="new Vue({el:'#app'})")
r("JS-LEGACY-LIBS","Legacy JS libraries (Dojo, ExtJS, Prototype, YUI, Backbone)","Frontend","Legacy JS",r"\bdojo\.(require|declare)\b|\bExt\.(define|create|onReady)\b|\bnew\s+Ajax\.(Request|Updater)\b|\bYAHOO\.|\bYUI\(|\bBackbone\.(Model|View)\b",JS,"medium",match="dojo.require('dijit.form.Button');",tags=["legacy"])
r("JS-FLASH","Flash / Flex (end of life)","Frontend","Flash/Flex",r"\.swf\b|application/x-shockwave-flash|\bmx:Application\b",JS+["ActionScript/Flex"],"high","high",match="<embed src=\"app.swf\">",tags=["eol"])
r("JS-FLASH-FILE","Flash binary committed","Frontend","Flash/Flex",path=r"\.swf$",sev="high",tags=["eol"])
r("JS-NODE-SERVER","Node.js server","Web","Node.js",r"require\(\s*['\"](express|koa|hapi|fastify)['\"]\s*\)|from\s+['\"](express|@nestjs/core|fastify)['\"]",["JavaScript","TypeScript"],match="const express = require('express');")

# ---------- Python / PHP ----------
r("PY-WEB","Python web framework","Web","Python web",r"(?m)^\s*(from|import)\s+(django|flask|fastapi|tornado|pyramid|bottle)\b",["Python"],match="from flask import Flask")
r("PY-ORM","Python ORM","Persistence","Python ORM",r"(?m)^\s*(from|import)\s+(sqlalchemy|peewee|pony|django\.db)\b",["Python"],match="from sqlalchemy import create_engine")
r("PY-CELERY","Celery tasks","Scheduling","Celery",r"(?m)^\s*(from|import)\s+celery\b",["Python"],match="from celery import Celery")
r("PY2-SYNTAX","Python 2 syntax (end of life)","Migration","Python 2",r"(?m)^\s*print\s+[\"'\w]|\bxrange\s*\(|\braw_input\s*\(|\bexcept\s+\w+\s*,\s*\w+\s*:|\.has_key\s*\(",["Python"],"high","medium",match="print 'hola'",nomatch="print('hola')",tags=["eol"])
r("PHP-FRAMEWORK","PHP framework","Web","PHP framework",r"\bIlluminate\\|\bSymfony\\Component\b|\bZend\\|\bCodeIgniter\b|\bYii::",["PHP"],match="use Illuminate\\Support\\Facades\\DB;")
r("PHP-MYSQL-LEGACY","Legacy mysql_* API (removed in PHP 7)","Migration","PHP mysql_*",r"\bmysql_(connect|pconnect|query|fetch_\w+|select_db|real_escape_string)\s*\(",["PHP"],"high","high",match="$r = mysql_query($q);",nomatch="$r = mysqli_query($c,$q);",tags=["eol"])

# ---------- Authentication & transport security ----------
r("AUTH-SPRING-METHOD","Method-level authorization","Authentication","Spring Security",r"@(PreAuthorize|PostAuthorize|Secured|RolesAllowed)\b",JVM,match="@PreAuthorize(\"hasRole('ADMIN')\")",nomatch="// @Secured",tags=["auth"])
r("AUTH-X509","X.509 / client-certificate authentication (mTLS)","Authentication","X.509 client auth",r"\bX509AuthenticationFilter\b|\.x509\s*\(|\bCLIENT[_-]CERT\b|\bclientAuth\b|\bExtractPrincipalFromCertificate\b",JVM+CONF,"info",match=["http.x509();","auth-method CLIENT-CERT"],nomatch="// x509 notes",msg="Client-certificate authentication implies mutual TLS (mTLS).",tags=["auth","mtls"])
r("AUTH-JWT","JSON Web Token authentication","Authentication","JWT",r"\bio\.jsonwebtoken\b|\bcom\.nimbusds\.jwt\b|\bcom\.auth0\.jwt\b|\bJwtDecoder\b|\bBearerTokenAuthentication\b|\bOncePerRequestFilter\b.{0,80}[Jj][Ww][Tt]",JVM+["Maven POM","Gradle"],match="import io.jsonwebtoken.Jwts;",nomatch="// jwt todo",tags=["auth","token"])
r("AUTH-OAUTH2","OAuth2 / OpenID Connect","Authentication","OAuth2/OIDC",r"\borg\.springframework\.security\.oauth2\b|@Enable(Authorization|Resource)Server\b|\bspring-security-oauth\b|\bClientRegistration\b|\bOAuth2(Login|Client|ResourceServer|AuthorizedClient)\b",JVM+["Maven POM","Gradle"],match="import org.springframework.security.oauth2.jwt.JwtDecoder;",tags=["auth","token"])
r("AUTH-SAML","SAML 2.0 federation","Authentication","SAML",r"\bopensaml\b|\borg\.springframework\.security\.saml\b|\bsaml2Login\b|urn:oasis:names:tc:SAML",JVM+CONF,match="import org.opensaml.saml.saml2.core.Assertion;",tags=["auth","sso"])
r("AUTH-CAS","CAS single sign-on","Authentication","CAS",r"\borg\.(jasig|apereo)\.cas\b|\bcas-client\b|\bCasAuthenticationFilter\b",JVM+CONF,match="import org.jasig.cas.client.validation.Cas20ServiceTicketValidator;",tags=["auth","sso"])
r("AUTH-KEYCLOAK","Keycloak adapter","Authentication","Keycloak",r"\borg\.keycloak\b|\bkeycloak\.json\b|\bKeycloakSecurityContext\b",JVM+CONF+["OpenAPI/JSON"],match="import org.keycloak.adapters.KeycloakDeployment;",tags=["auth","sso"])
r("AUTH-KERBEROS","Kerberos / SPNEGO","Authentication","Kerberos/SPNEGO",r"\borg\.ietf\.jgss\b|\bGSSManager\b|\bSPNEGO\b|\bKrb5LoginModule\b|\bspring-security-kerberos\b",JVM+CONF,match="import org.ietf.jgss.GSSManager;",tags=["auth","sso"])
r("AUTH-LDAP","LDAP / Active Directory authentication","Authentication","LDAP/AD",r"\bLdapContext\b|\bInitialLdapContext\b|\bspring-security-ldap\b|\bActiveDirectoryLdapAuthenticationProvider\b|\bjavax\.naming\.directory\b",JVM+CONF,match="import javax.naming.directory.InitialDirContext;",tags=["auth"])
r("AUTH-WS-SECURITY","WS-Security (SOAP message security)","Authentication","WS-Security",r"\bWSS4J(In|Out)?Interceptor\b|\borg\.apache\.wss4j\b|\bws-security\b|oasis-200401-wss-wssecurity|\bUsernameToken\b",JVM+CONF,match="import org.apache.cxf.ws.security.wss4j.WSS4JInInterceptor;",msg="Message-level security on SOAP services; check tokens, signature and encryption.",tags=["auth","soap"])
r("AUTH-APIKEY","API key authentication","Authentication","API key",r"\bX-API-KEY\b|\bOcp-Apim-Subscription-Key\b|\bapi[_-]?key\b",CODE+CONF,"low",match="request.getHeader(\"X-API-KEY\");",nomatch="// no api-key here",tags=["auth"])
r("AUTH-BASIC-OUTBOUND","HTTP Basic credentials in an outbound client","Authentication","HTTP Basic",r"\"Authorization\"\s*,\s*\"Basic\s|\bUsernamePasswordCredentials\b|\bsetBasicAuth\b|\bPreemptiveBasicAuth\b",CODE,"low",match="conn.setRequestProperty(\"Authorization\", \"Basic \"+tok);",nomatch="// basic auth",tags=["auth","outbound"])
r("AUTH-BEARER-OUTBOUND","Bearer token in an outbound client","Authentication","Bearer token",r"\"Authorization\"\s*,\s*\"Bearer\s|\.setBearerAuth\b",CODE,match="h.set(\"Authorization\", \"Bearer \"+t);",tags=["auth","outbound"])
# Inbound mTLS terminated by an intermediary (proxy, API gateway, load balancer): the application only
# sees an HTTP request with the client certificate, or data from it, in a header. Rules whose id starts
# with TLS-UPSTREAM- mark the endpoints of their file as "mTLS at an intermediary (inferred)".
r("TLS-UPSTREAM-CERT-HEADER","Client certificate received in an HTTP header (mTLS terminated by an intermediary)","Transport Security","mTLS at intermediary (inferred)",r"""\b(?:getHeaders?|getHeaderString|getRequestHeader)\s*\(\s*["'][^"']*(?i:cert|x509)[^"']*["']|\b(?:[Hh]eaders|META|req|request|Request)\.(?:get|Get|header|TryGetValue|GetValues)\s*\(\s*["'][^"']*(?i:cert|x509)[^"']*["']|@(?:RequestHeader|HeaderParam)\s*\(\s*(?:(?:value|name)\s*=\s*)?"[^"]*(?i:cert|x509)[^"]*"|\[FromHeader\s*\(\s*Name\s*=\s*"[^"]*(?i:cert|x509)[^"]*"|[Hh]eaders\s*\[\s*["'][^"']*(?i:cert|x509)[^"']*["']\s*\]|["'](?i:ssl[_-]client[_-]cert|x-ssl-client-cert|x-client-cert(?:ificate)?|x-forwarded-client-cert|x-arr-clientcert|x-ssl-cert|x-amzn-mtls-clientcert)["']|["']HTTP_[A-Z0-9_]*(?:CERT|X509)[A-Z0-9_]*["']""",CODE,"info","medium",match=["String pem = request.getHeader(\"X-Client-Cert\");","public Response sign(@HeaderParam(\"SSL_CLIENT_CERT\") String cert) {","@RequestHeader(value = \"x-certificado\") String certificado","var c = Request.Headers[\"X-ARR-ClientCert\"];","cert = request.META.get(\"HTTP_X_SSL_CLIENT_CERT\")","const pem = req.header('x-client-cert');","private static final String CERT_HEADER = \"X-Forwarded-Client-Cert\";"],nomatch=["String ct = request.getHeader(\"Content-Type\");","// String pem = request.getHeader(\"X-Client-Cert\");","String certPath = \"/etc/app/cert.pem\";","String p = props.get(\"cert.path\");","var c = config.get(\"ssl.cert\");"],msg="The application reads the client certificate from a header: an intermediary (proxy, API gateway, load balancer) terminates mutual TLS in front of it. Who requires it and which consumers use it is not in the code.",tags=["mtls","inferred"])
r("TLS-UPSTREAM-CERT-DATA-HEADER","Client certificate data received in an HTTP header (mTLS terminated by an intermediary)","Transport Security","mTLS at intermediary (inferred)",r"""["'](?i:ssl[_-]client[_-](?:s[_-]dn|i[_-]dn|serial|verify|fingerprint)|x-ssl-client-(?:s-)?dn|x-ssl-client-(?:verify|serial|fingerprint)|x-client-(?:dn|subject|serial|verify|fingerprint))["']|["']HTTP_SSL_CLIENT_(?:S_DN|I_DN|VERIFY|SERIAL)["']""",CODE,"info","low",match=["String dn = request.getHeader(\"SSL_CLIENT_S_DN\");","if (\"SUCCESS\".equals(req.getHeader(\"X-SSL-Client-Verify\")))"],nomatch=["String dn = request.getHeader(\"X-Request-Id\");"],msg="Subject, serial number or verification result of a client certificate arrive in a header: likely mutual TLS terminated by an intermediary.",tags=["mtls","inferred"])
r("TLS-UPSTREAM-CERT-HEADER-CONFIG","Configured header carrying the client certificate","Transport Security","mTLS at intermediary (inferred)",r"""(?m)^\s*[\w.\-]*(?i:cert|x509)[\w.\-]*(?i:header)[\w.\-]*\s*[:=]\s*\S+|^\s*[\w.\-]*(?i:header)[\w.\-]*\s*[:=]\s*["']?[\w\-]*(?i:cert|x509)[\w\-]*""",CONF,"info","low",match=["security.cert-header=X-Client-Cert","auth.header.name: SSL_CLIENT_CERT"],nomatch=["server.ssl.key-store=keystore.p12","http.header.timeout=30"],msg="A header that carries the client certificate is configured: mutual TLS is likely terminated by an intermediary.",tags=["mtls","inferred"])
r("TLS-CLIENT-IDENTITY","Client TLS identity (keystore) — mutual TLS","Transport Security","mTLS client",r"\bKeyManagerFactory\b|\bloadKeyMaterial\b|\.setKeyManagers\b|javax\.net\.ssl\.keyStore\b|\bTLSClientParameters\b",JVM+CONF,"info",match="KeyManagerFactory kmf = KeyManagerFactory.getInstance(\"SunX509\");",nomatch="// keystore",msg="A client keystore configures mutual TLS (client-certificate) on outbound calls.",tags=["mtls","outbound"])
r("TLS-CUSTOM-CONTEXT","Custom TLS/SSL context","Transport Security","Custom TLS",r"\bSSLContext\b|\bSSLSocketFactory\b|\bSslContextBuilder\b",JVM,"info",match="SSLContext ctx = SSLContext.getInstance(\"TLS\");",tags=["tls"])
r("TLS-VERIFICATION-DISABLED","TLS verification disabled","Transport Security","Insecure TLS",r"\bALLOW_ALL_HOSTNAME_VERIFIER\b|\bNoopHostnameVerifier\b|\btrustAllCerts\b|\bTrustAllStrategy\b|\bsetHostnameVerifier\s*\([^)]*ALLOW_ALL|checkServerTrusted\s*\([^)]*\)\s*\{\s*\}",JVM,"high","high",match="SSLContexts.custom().loadTrustMaterial(null, new TrustAllStrategy());",nomatch="// verify certs properly",msg="Certificate or hostname verification is bypassed: traffic is not authenticated.",tags=["security","tls"])
r("TLS-WEAK-PROTOCOL","Weak TLS/SSL protocol","Transport Security","Weak TLS",r"\b(SSLv2|SSLv3|TLSv1\.0|TLSv1\.1)\b|\"TLSv1\"|\"SSLv3\"",JVM+CONF,"medium","medium",match="ctx = SSLContext.getInstance(\"TLSv1\");",nomatch="ctx = SSLContext.getInstance(\"TLSv1.2\");",tags=["security","tls"])
r("TLS-SPRING-BOOT-MTLS","Spring Boot client-certificate (mutual TLS)","Authentication","X.509 client auth",r"server\.ssl\.client-auth\s*[=:]\s*(need|want)|\bclient-auth:\s*(need|want)\b",CONF,"info",match="server.ssl.client-auth=need",nomatch="server.ssl.enabled=true",tags=["mtls"])
r("TLS-SPRING-BOOT","Spring Boot TLS keystore","Transport Security","TLS",r"server\.ssl\.key-store\b|server\.ssl\.enabled\s*[=:]\s*true",CONF,"info",match="server.ssl.key-store=classpath:ks.p12",nomatch="server.port=8080",tags=["tls"])
r("AUTH-CUSTOM-PROVIDER","Custom login: own authentication provider or filter","Authentication","Custom authentication",r"\bimplements\s+[\w\s,.<>]*\b(AuthenticationProvider|UserDetailsService|AuthenticationManager)\b|\bextends\s+(AbstractAuthenticationProcessingFilter|AbstractUserDetailsAuthenticationProvider|UsernamePasswordAuthenticationFilter|AbstractAuthenticationToken)\b",JVM,"info",match="public class AcmeAuthenticationProvider implements AuthenticationProvider {",nomatch="private AuthenticationProvider provider;",msg="User login implemented by the application itself (the class named in the evidence).",tags=["auth","user"])
r("AUTH-PREAUTH-SSO","Pre-authenticated SSO (login done by a gateway or the container)","Authentication","Pre-authenticated SSO",r"\b(RequestHeaderAuthenticationFilter|AbstractPreAuthenticatedProcessingFilter|J2eePreAuthenticatedProcessingFilter|PreAuthenticatedAuthenticationProvider|WebSphereRequestHeaderAuthenticationFilter)\b|getHeader\s*\(\s*\"(OIDC_CLAIM_\w*|X-Forwarded-User|X-Remote-User|REMOTE_USER|iv-user|SM_USER|X-Auth-User|X-Auth-Request-User)\"",JVM,"info",match="new RequestHeaderAuthenticationFilter()",nomatch="// RequestHeaderAuthenticationFilter",msg="The application trusts an identity established upstream (reverse proxy, SSO gateway or container).",tags=["auth","user","sso"])
r("AUTH-CONTAINER","Container-managed authentication","Authentication","Container (JAAS)",r"\bgetUserPrincipal\s*\(\s*\)|\bgetRemoteUser\s*\(\s*\)|<security-domain>|<auth-method>\s*(FORM|BASIC|DIGEST)\s*</auth-method>|<application-policy\b",JVM+CONF,"info",match=["String u = request.getRemoteUser();","<security-domain>orders</security-domain>"],nomatch="String u = getUser();",msg="Login is configured in the application server (security domain / realm), outside the application code.",tags=["auth","user"])
r("AUTH-KEYCLOAK-CONFIG","Keycloak adapter configuration","Authentication","Keycloak",path=r"(?i)(^|/)(keycloak\.json|keycloak-saml\.xml|keycloak-oidc\.json)$",tags=["auth","sso","user"])
r("AUTH-KEYCLOAK-METHOD","Keycloak authentication method","Authentication","Keycloak",r"<auth-method>\s*KEYCLOAK(-SAML)?\s*</auth-method>|\bKeycloakWebSecurityConfigurerAdapter\b|\bKeycloakAuthenticationProvider\b",JVM+CONF,"info",match="<auth-method>KEYCLOAK-SAML</auth-method>",tags=["auth","sso","user"])
r("TLS-LIBERTY-MTLS","Liberty TLS with client certificates (mTLS)","Authentication","X.509 client auth",r"<ssl\b[^>]*\bclientAuthentication(Supported)?\s*=\s*\"true\"",["XML"],"info",match="<ssl id=\"def\" keyStoreRef=\"ks\" clientAuthentication=\"true\"/>",nomatch="<ssl id=\"def\" keyStoreRef=\"ks\"/>",msg="Open Liberty / WebSphere Liberty requires (or accepts) client certificates.",tags=["mtls"])
r("PLAT-LIBERTY-SECURITY","Liberty security features","Authentication","Liberty security",r"<feature>\s*(appSecurity|transportSecurity|openidConnectClient|openidConnectServer|samlWeb|socialLogin|jwt|mpJwt|ldapRegistry|spnego)-[\d.]+\s*</feature>",["XML"],"info",match="<feature>samlWeb-2.0</feature>",nomatch="<feature>servlet-4.0</feature>",tags=["auth","liberty"])

# ---------- Electronic signature & federated identity (standards, not countries) ----------
r("SIGN-XMLDSIG","XML Signature (XML-DSig)","Authentication","XML-DSig",r"\bjavax\.xml\.crypto\.dsig\b|\borg\.apache\.xml\.security\.signature\b|\bXMLSignatureFactory\b|\bSystem\.Security\.Cryptography\.Xml\b|\bSignedXml\b|\bsignxml\b",JVM+NET+["Python"],match=["import javax.xml.crypto.dsig.XMLSignatureFactory;","var signed = new SignedXml(doc);"],nomatch="// XMLSignatureFactory",tags=["signature"])
r("SIGN-CMS","CMS / PKCS#7 signature","Authentication","CMS/PKCS#7",r"\borg\.bouncycastle\.cms\b|\bCMSSignedData(Generator)?\b|\bSignedCms\b|\bSystem\.Security\.Cryptography\.Pkcs\b",JVM+NET,match=["import org.bouncycastle.cms.CMSSignedDataGenerator;","var cms = new SignedCms(content);"],nomatch="String cms = \"page\";",tags=["signature"])
r("SIGN-PDF","PDF digital signature","Authentication","PDF signature",r"\borg\.apache\.pdfbox\.pdmodel\.interactive\.digitalsignature\b|\bPDSignature\b|\bPdfSigner\b|\bPdfSignatureAppearance\b|\bPdfPKCS7\b",JVM+NET,match=["import org.apache.pdfbox.pdmodel.interactive.digitalsignature.PDSignature;","PdfSigner signer = new PdfSigner(reader, out, props);"],tags=["signature"])
r("SIGN-ADES","Advanced electronic signature (XAdES / CAdES / PAdES / JAdES)","Authentication","AdES",r"\beu\.europa\.esig\.dss\b|\b[XCPJ]AdES(Service|Signature|SignatureParameters|_BASELINE_[BTLA]+)?\b|\bxades4j\b",JVM+CONF,match=["import eu.europa.esig.dss.xades.signature.XAdESService;","SignatureLevel.PAdES_BASELINE_B"],nomatch="String cades = \"x\";",tags=["signature"])
r("SIGN-TIMESTAMP","Trusted timestamping (RFC 3161 / TSA)","Authentication","Timestamping",r"\bRFC\s?3161\b|\bTimeStampToken\b|\bTimeStampRequest\b|\borg\.bouncycastle\.tsp\b|\bTSAClient\b",JVM,match="import org.bouncycastle.tsp.TimeStampToken;",tags=["signature","pki"])
r("SIGN-PKCS11","Hardware token / smart card (PKCS#11)","Authentication","PKCS#11",r"\bSunPKCS11\b|\bPKCS11(Provider|Config)?\b|\bpkcs11\.cfg\b|\bIAIK\b.{0,20}\bpkcs11\b",JVM+CONF,match=["Provider p = Security.getProvider(\"SunPKCS11\");"],nomatch="int pkcs = 11;",tags=["signature","pki"])
r("AUTH-EIDAS","eIDAS cross-border identity (SAML profile)","Authentication","eIDAS",r"\beIDAS\b|\bSTORK\b|\beu\.eidas\b",JVM+CONF+JS,match="import eu.eidas.auth.commons.EidasStringUtil;",nomatch="String eidasish = \"x\";",tags=["sso"])

# ---------- Contracts, delivery, infrastructure ----------
r("API-OPENAPI","OpenAPI / Swagger contract","Contract","OpenAPI",r"(?m)^\s*\"?(openapi|swagger)\"?\s*:\s*[\"']?[23]\.|@OpenAPIDefinition\b|\bio\.swagger\b",["YAML","OpenAPI/JSON","Java","Kotlin"],match="openapi: 3.0.1")
r("API-WSDL","WSDL contract file","Contract","WSDL",path=r"\.wsdl$")
r("API-PROTO","gRPC / Protobuf contract","Contract","gRPC",path=r"\.proto$")
r("OPS-DOCKER","Container image definition","Delivery","Docker",path=r"(^|/)(Dockerfile|Containerfile)[^/]*$")
r("OPS-K8S","Kubernetes manifests","Delivery","Kubernetes",r"(?m)^kind:\s*(Deployment|StatefulSet|DaemonSet|Service|Ingress|ConfigMap|CronJob|Route)\b",["YAML"],match="kind: Deployment")
r("OPS-HELM","Helm chart","Delivery","Helm",path=r"(^|/)Chart\.yaml$")
r("OPS-GITLAB-CI","GitLab CI pipeline","Delivery","GitLab CI",path=r"(^|/)\.gitlab-ci\.yml$")
r("OPS-JENKINS","Jenkins pipeline","Delivery","Jenkins",path=r"(^|/)Jenkinsfile$")
r("OPS-ANT-BUILD","Ant build (no dependency management)","Delivery","Ant",path=r"(^|/)build\.xml$",sev="low",tags=["legacy"])
r("OPS-VENDORED-LIBS","Libraries committed to the repository","Delivery","Vendored libraries",path=r"(^|/)(WEB-INF/lib|lib|libs)/[^/]+\.jar$",sev="low",msg="Dependencies without a manager: versions and CVEs are hard to track.",tags=["supply-chain"])
r("OPS-INSECURE-REPO","Artifact repository over plain HTTP","Security","Build",r"(?s)<(repository|pluginRepository|snapshotRepository|mirror)>.{0,400}?<url>\s*http://|\bmaven\s*\{\s*url\s*[=(]?\s*['\"]http://",["Maven POM","Gradle","XML"],"medium",match=["<repository><id>x</id><url>http://repo.example.org/maven</url></repository>","maven { url 'http://repo.example.org' }"],nomatch="<license><url>http://www.apache.org/licenses/LICENSE-2.0</url></license>",msg="Dependencies downloaded without TLS can be tampered with.",tags=["security","supply-chain"])
r("OPS-SHELL-SCRIPTS","Operational shell / batch scripts","Delivery","Scripts",path=r"\.(sh|ksh|bat|cmd)$")

# How each authentication-related rule is presented (user login, service to
# service, e-signature, transport). Rules not listed derive it from the category.
KIND = {
 "user": ["AUTH-CUSTOM-PROVIDER","AUTH-PREAUTH-SSO","AUTH-CONTAINER","JAVA-JAAS","JAVA-SPRING-SECURITY","AUTH-SPRING-METHOD",
          "AUTH-OAUTH2","AUTH-SAML","AUTH-CAS","AUTH-KEYCLOAK","AUTH-KEYCLOAK-CONFIG","AUTH-KEYCLOAK-METHOD","AUTH-KERBEROS",
          "AUTH-LDAP","AUTH-JWT","AUTH-EIDAS","PLAT-LIBERTY-SECURITY"],
 "service": ["AUTH-APIKEY","AUTH-BASIC-OUTBOUND","AUTH-BEARER-OUTBOUND","AUTH-WS-SECURITY"],
 "signature": ["SIGN-XMLDSIG","SIGN-CMS","SIGN-PDF","SIGN-ADES","SIGN-TIMESTAMP","SIGN-PKCS11"],
 "transport": ["AUTH-X509","TLS-CLIENT-IDENTITY","TLS-CUSTOM-CONTEXT","TLS-VERIFICATION-DISABLED","TLS-WEAK-PROTOCOL",
               "TLS-SPRING-BOOT-MTLS","TLS-SPRING-BOOT","TLS-LIBERTY-MTLS",
               "TLS-UPSTREAM-CERT-HEADER","TLS-UPSTREAM-CERT-DATA-HEADER","TLS-UPSTREAM-CERT-HEADER-CONFIG"],
}
def apply_kinds(rules):
    ids={d["id"] for d in rules}
    for k,v in KIND.items():
        for i in v:
            assert i in ids, "authKind for unknown rule "+i
    for d in rules:
        for k,v in KIND.items():
            if d["id"] in v: d["authKind"]=k
apply_kinds(R)

json.dump({"version":"2.0.0","rules":R},open(sys.argv[1],"w"),indent=1,ensure_ascii=False)
print(len(R),"rules")

# ---------- Optional rule packs (not embedded; loaded with --rules or "rules" in traverse.json) ----------
# Regional or sector knowledge lives here, outside the agnostic core catalog.
if len(sys.argv) > 2:
    import os
    R = []
    r("ES-AFIRMA","@firma / AutoFirma e-signature client (Spain)","Authentication","@firma/AutoFirma",r"\bes\.gob\.afirma\b|\bAutoFirma\b|\b@firma\b|\bMiniApplet\b|\bafirma[._-]?signer\b",JVM+CONF+JS,match="import es.gob.afirma.core.signers.AOSigner;",nomatch="// afirma",msg="Spanish public-sector e-signature platform.",tags=["signature","es"])
    r("ES-CLAVE","Cl@ve identity (Spain)","Authentication","Cl@ve",r"\bes\.gob\.clave\b|\bcl@ve\b|\bminhafp.*clave\b",JVM+CONF+JS,match="import es.gob.clave.ClaveSP;",nomatch="String clave = getPassword();",msg="Spanish public-sector federated identity.",tags=["sso","es"])
    r("ES-FIRE","FIRe / Cl@ve Firma central signature (Spain)","Authentication","FIRe",r"\bes\.gob\.fire\b|\bFIReClient\b|\bclavefirma\b|\bfire[._-]?client\b",JVM+CONF,match="import es.gob.fire.client.FireClient;",msg="Spanish public-sector central signature service.",tags=["signature","es"])
    r("ES-VALIDA","@firma Valida certificate validation (Spain)","Authentication","Valida",r"\bDSSAfirmaVerify\b|\beVisor\b|\bvalida\b.{0,30}certificad|\bafirma\b.{0,20}valida",JVM+CONF,match="DSSAfirmaVerify verify = new DSSAfirmaVerify();",tags=["signature","es"])
    r("ES-FNMT","FNMT / CERES certificate authority (Spain)","Authentication","FNMT",r"\bFNMT\b|\bCERES\b|\bAC\s?FNMT\b",JVM+CONF,match="return CertProvider.FNMT;",nomatch="int fnmtp = 0;",tags=["pki","es"])
    for d in R:
        d["authKind"] = "user" if d["id"] == "ES-CLAVE" else "signature"
    out = os.path.join(sys.argv[2], "es-public-sector.json")
    json.dump({"version":"es-public-sector-1.0.0","rules":R},open(out,"w"),indent=1,ensure_ascii=False)
    print(len(R),"rules in",out)
