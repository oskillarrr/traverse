package yamlite

import "testing"

const k8s = `# deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: orders   # trailing comment
  labels: {app: orders, tier: "backend"}
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: orders
    spec:
      containers:
      - name: orders
        image: "registry.example.org:5000/team/orders:1.4.2"
        ports:
        - containerPort: 8080
        env:
          - name: DB_URL
            valueFrom:
              secretKeyRef: {name: db, key: url}
          - name: "MODE"
            value: 'prod'
        command: ["java", "-jar", "app.jar"]
        args:
          - |
            multi
            line
---
apiVersion: v1
kind: Service
metadata: {name: orders}
spec:
  selector:
    app: orders
  ports:
  - port: 80
    targetPort: 8080
`

func TestKubernetes(t *testing.T) {
	docs, err := ParseAll(k8s)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("docs = %d", len(docs))
	}
	d := docs[0]
	if d.S("kind") != "Deployment" || d.S("metadata", "name") != "orders" || d.S("spec", "replicas") != "3" {
		t.Errorf("basic fields: %q %q %q", d.S("kind"), d.S("metadata", "name"), d.S("spec", "replicas"))
	}
	if d.S("metadata", "labels", "tier") != "backend" {
		t.Errorf("flow map")
	}
	c := d.Path("spec", "template", "spec", "containers").List()
	if len(c) != 1 || c[0].S("image") != "registry.example.org:5000/team/orders:1.4.2" {
		t.Fatalf("containers: %+v", c)
	}
	env := c[0].Get("env").List()
	if len(env) != 2 || env[1].S("name") != "MODE" || env[1].S("value") != "prod" || env[0].S("valueFrom", "secretKeyRef", "key") != "url" {
		t.Errorf("env: %+v", env)
	}
	if len(c[0].Get("command").List()) != 3 || c[0].Get("ports").List()[0].S("containerPort") != "8080" {
		t.Errorf("flow seq / ports")
	}
	if docs[1].S("metadata", "name") != "orders" || docs[1].Path("spec", "ports").List()[0].S("targetPort") != "8080" {
		t.Errorf("service")
	}
}

const compose = `version: "3.8"
services:
  web:
    build: ./web
    ports:
      - "8080:8080"
    depends_on:
      - db
    environment:
      SPRING_DATASOURCE_URL: jdbc:postgresql://db:5432/app
  db:
    image: postgres:9.6
    environment:
      - POSTGRES_PASSWORD=example
`

func TestCompose(t *testing.T) {
	docs, _ := ParseAll(compose)
	s := docs[0].Get("services")
	if got := s.Keys(); len(got) != 2 || got[0] != "web" {
		t.Fatalf("services: %v", got)
	}
	if s.S("web", "build") != "./web" || s.Path("web", "depends_on").List()[0].Value != "db" {
		t.Errorf("web")
	}
	if s.Path("db", "environment").List()[0].Value != "POSTGRES_PASSWORD=example" {
		t.Errorf("env list")
	}
}

func TestHelmSafe(t *testing.T) {
	src := "{{- if .Values.enabled }}\nkind: Deployment\nmetadata:\n  name: {{ include \"x.fullname\" . }}\n{{- end }}\n"
	docs, err := ParseAll(HelmSafe(src))
	if err != nil || docs[0].S("metadata", "name") != "TEMPLATE" {
		t.Fatalf("helm: %v %+v", err, docs)
	}
}
