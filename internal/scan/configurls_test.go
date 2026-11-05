package scan

import (
	"testing"

	"github.com/oskillarrr/traverse/internal/lang"
	"github.com/oskillarrr/traverse/internal/model"
)

func cfg(t *testing.T, rel, text string) []model.Contract {
	t.Helper()
	l, _ := lang.Detect(rel)
	return configContracts(rel, l, text)
}

func TestConfigURLsProperties(t *testing.T) {
	cs := cfg(t, "fire_config/config.properties", `# comentario https://no.es/x
signer.url=https://signer.example.es/signerws/services/DSSSignerVerify
pasarela.taxgw.endpoint = http://pasarela.example.es/custodia
mail.smtp.host=smtp.example.es
datasource.url: jdbc:oracle:thin:@//db.example.es:1521/ORDERSPRO
test.url=https://mock.example.es/x
vacia=${SERVICIO_URL}
`)
	byCp := map[string]model.Contract{}
	for _, c := range cs {
		byCp[c.Counterpart] = c
	}
	if c := byCp["signer"]; c.Kind != "http" || c.Transport != "tls" || c.Match != "/signerws/services/DSSSignerVerify" || c.Host != "signer.example.es" || c.Line != 2 {
		t.Errorf("signer: %+v", c)
	}
	if c := byCp["pasarela.taxgw"]; c.Transport != "plain" || c.Line != 3 {
		t.Errorf("pasarela: %+v", c)
	}
	if c := byCp["mail"]; c.Kind != "mail" || c.Host != "smtp.example.es" {
		t.Errorf("mail: %+v", c)
	}
	if c := byCp["database"]; c.Kind != "database" || c.Key != "oracle" || c.Host != "db.example.es/ORDERSPRO" {
		t.Errorf("jdbc: %+v", c)
	}
	if len(cs) != 5 {
		t.Errorf("want 5 contracts (comment and placeholder ignored), got %d: %+v", len(cs), cs)
	}
	for _, c := range cs {
		if c.Detail == "configuration: test.url" && c.Confidence != "low" {
			t.Errorf("test key should be low confidence: %+v", c)
		}
	}
}

func TestConfigURLsYAMLAndXML(t *testing.T) {
	ys := cfg(t, "application.yml", `payroll:
  ws:
    endpoint: "https://payroll.example.es/wsTurnosPayroll/serviciosTurnos?wsdl"
spring:
  datasource:
    url: jdbc:postgresql://pg.example.es:5432/orders
`)
	if len(ys) != 2 || ys[0].Counterpart != "payroll" || ys[0].Kind != "soap" || ys[1].Host != "pg.example.es/orders" {
		t.Fatalf("yaml: %+v", ys)
	}
	xs := cfg(t, "WEB-INF/spring-ws.xml", `<?xml version="1.0"?>
<beans xmlns="http://www.springframework.org/schema/beans" xsi:schemaLocation="http://x.org/a http://x.org/a.xsd">
  <!-- <property name="old" value="https://old.example.es/x"/> -->
  <bean id="ledgerClient" class="x.Y">
    <property name="url" value="https://ledger.example.es/ledger/ws"/>
  </bean>
  <entry key="portal.base">https://portal.example.org/portal</entry>
</beans>`)
	if len(xs) != 2 || xs[0].Counterpart != "ledger" || xs[1].Counterpart != "portal" {
		t.Fatalf("xml: %+v", xs)
	}
	st := cfg(t, "standalone.xml", `<server>
 <datasource jndi-name="java:/jdbc/ordersDS"><connection-url>jdbc:oracle:thin:@(DESCRIPTION=(ADDRESS=(PROTOCOL=TCPS)(HOST=ora.example.es)(PORT=2484))(CONNECT_DATA=(SERVICE_NAME=ORDERS)))</connection-url></datasource>
 <outbound-socket-binding name="mail-smtp"><remote-destination host="smtp.example.es" port="25"/></outbound-socket-binding>
</server>`)
	if len(st) != 2 || st[0].Kind != "database" || st[0].Transport != "tls" || st[0].Host != "ora.example.es/ORDERS" || st[1].Kind != "mail" {
		t.Fatalf("standalone: %+v", st)
	}
}

func TestCounterpartFromKey(t *testing.T) {
	for k, want := range map[string]string{"signer.url": "signer", "pasarela.taxgw.endpoint": "pasarela.taxgw",
		"payroll.ws.endpoint": "payroll", "ledgerClient.url": "ledger", "url": "", "app.crm.base-url": "crm"} {
		if got := counterpartFromKey(k); got != want {
			t.Errorf("%s → %q, want %q", k, got, want)
		}
	}
}
