package lang

import (
	"strings"
	"testing"
)

func TestStripPreservesOffsetsAndCounts(t *testing.T) {
	src := "package a;\n// import javax.jms.Queue;\n/* block\n   comment */\nString s = \"// not a comment\";\n\n"
	st := Strip(src, CStyle)
	if len(st.Code) != len(src) || len(st.Bare) != len(src) {
		t.Fatal("length changed")
	}
	if strings.Contains(st.Code, "javax.jms") || strings.Contains(st.Code, "block") {
		t.Errorf("comments not removed: %q", st.Code)
	}
	if !strings.Contains(st.Code, `"// not a comment"`) {
		t.Errorf("string literal must be kept in Code")
	}
	if strings.Contains(st.Bare, "not a comment") {
		t.Errorf("string content must be blank in Bare")
	}
	if st.CodeN != 2 || st.CommN != 3 || st.BlankN != 1 {
		t.Errorf("counts code=%d comment=%d blank=%d", st.CodeN, st.CommN, st.BlankN)
	}
	if l := st.Line(strings.Index(src, "String")); l != 5 {
		t.Errorf("line = %d", l)
	}
}

func TestCOBOLFixedFormat(t *testing.T) {
	src := "       IDENTIFICATION DIVISION.\n      * EXEC CICS in a comment\n           EXEC CICS RETURN END-EXEC.\n"
	st := Strip(src, COBOL)
	if strings.Count(st.Code, "EXEC CICS") != 1 || st.CommN != 1 {
		t.Errorf("COBOL comment handling: %q", st.Code)
	}
}

func TestUnterminatedCharLiteralRecovers(t *testing.T) {
	st := Strip("fn x<'a>() {}\nint y; // c\n", CStyle)
	if strings.Contains(st.Code, "// c") {
		t.Errorf("lexer did not recover after unterminated literal")
	}
}

func TestDetect(t *testing.T) {
	for p, want := range map[string]string{"a/pom.xml": "Maven POM", "X.java": "Java", "b/NIGHTLY.cbl": "COBOL",
		"Dockerfile.prod": "Dockerfile", "w/Default.aspx": "ASP.NET", "requirements-dev.txt": "Python build"} {
		if l, ok := Detect(p); !ok || l.Name != want {
			t.Errorf("%s -> %v", p, l.Name)
		}
	}
}
