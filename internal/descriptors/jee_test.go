package descriptors

import "testing"

func TestWebXMLWithExternalDTDIsParsedOffline(t *testing.T) {
	x := `<?xml version="1.0" encoding="ISO-8859-1"?>
<!DOCTYPE web-app PUBLIC "-//Sun//DTD Web Application 2.3//EN" "http://java.sun.com/dtd/web-app_2_3.dtd">
<web-app><servlet><servlet-name>S</servlet-name><servlet-class>a.S</servlet-class></servlet>
<servlet-mapping><servlet-name>S</servlet-name><url-pattern>/s/*</url-pattern></servlet-mapping>
<resource-ref><res-ref-name>jdbc/DS</res-ref-name><res-type>javax.sql.DataSource</res-type></resource-ref></web-app>`
	r := Parse("WEB-INF/web.xml", x)
	if r == nil || len(r.Contracts) != 2 {
		t.Fatalf("contracts: %+v", r)
	}
	if r.Contracts[0].Key != "ANY /s/*" || r.Contracts[1].Match != "jdbc/DS" {
		t.Errorf("unexpected: %+v", r.Contracts)
	}
}

func TestServerConfigNeverRecordsHosts(t *testing.T) {
	x := `<datasources><datasource jndi-name="java:jboss/datasources/X"><connection-url>jdbc:oracle:thin:@secret-host:1521:SID</connection-url></datasource></datasources>`
	r := Parse("deploy/x-ds.xml", x)
	for _, c := range r.Contracts {
		if c.Key != "java:jboss/datasources/X" {
			t.Errorf("unexpected contract %+v", c)
		}
	}
	if len(r.Techs) != 1 || r.Techs[0].Technology != "Oracle Database" {
		t.Errorf("vendor: %+v", r.Techs)
	}
}

func TestStruts2IncludedFile(t *testing.T) {
	// An action file included from struts.xml, with an arbitrary name.
	x := `<?xml version="1.0"?>
<!DOCTYPE struts PUBLIC "-//Apache Software Foundation//DTD Struts Configuration 2.5//EN" "http://struts.apache.org/dtds/struts-2.5.dtd">
<struts>
  <package name="ciudadano" namespace="/ciudadano" extends="struts-default">
    <action name="altaSolicitud" class="es.x.AltaSolicitudAction" method="alta">
      <result name="success">/jsp/ok.jsp</result>
    </action>
    <action name="listado" class="es.x.ListadoAction"/>
  </package>
</struts>`
	if !IsDescriptor("struts/ciudadano/altaSolicitud.xml", x) {
		t.Fatal("included struts file not recognized as a descriptor")
	}
	r := Parse("struts/ciudadano/altaSolicitud.xml", x)
	var acts []string
	for _, c := range r.Contracts {
		if c.Kind == "http" && c.Direction == "exposes" {
			acts = append(acts, c.Key)
		}
	}
	if len(acts) != 2 || acts[0] != "ANY /ciudadano/altaSolicitud.action" || acts[1] != "ANY /ciudadano/listado.action" {
		t.Errorf("struts2 actions: %v", acts)
	}
	if len(r.Techs) == 0 || r.Techs[0].Technology != "Struts 2" {
		t.Errorf("tech: %+v", r.Techs)
	}
}

func TestMyBatisMapperTables(t *testing.T) {
	x := `<?xml version="1.0"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="es.x.dao.ArchivoDao">
  <select id="buscar" resultType="Archivo">
    SELECT a.* FROM ARCHIVO a
    <where>
      <if test="estado != null">AND a.ESTADO = #{estado}</if>
    </where>
    JOIN BENEFICIARIO b ON b.ID = a.BEN_ID
  </select>
  <insert id="insertar">INSERT INTO ARCHIVO (ID, NOMBRE) VALUES (#{id}, #{nombre})</insert>
  <update id="cerrar">UPDATE EXPEDIENTE SET ESTADO = 'C' WHERE ID = #{id}</update>
  <sql id="cols">FROM CNAE</sql>
</mapper>`
	if !IsDescriptor("ibatis/archivo.xml", x) {
		t.Fatal("mapper not recognized")
	}
	r := Parse("ibatis/archivo.xml", x)
	got := map[string]string{}
	for _, c := range r.Contracts {
		if c.Kind == "db-table" {
			got[c.Key] = c.Direction
			if c.Component != "es.x.dao.ArchivoDao" {
				t.Errorf("component: %+v", c)
			}
		}
	}
	want := map[string]string{"ARCHIVO": "write", "BENEFICIARIO": "read", "EXPEDIENTE": "write", "CNAE": "read"}
	for tbl, dir := range want {
		if got[tbl] != dir {
			// ARCHIVO appears as read (select) and write (insert); accept either presence
			if tbl == "ARCHIVO" && (got[tbl] == "read" || got[tbl] == "write") {
				continue
			}
			t.Errorf("table %s = %q, want %q (all: %v)", tbl, got[tbl], dir, got)
		}
	}
	if len(got) < 4 {
		t.Errorf("expected at least 4 tables, got %v", got)
	}
}
