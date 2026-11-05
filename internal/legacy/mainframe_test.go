package legacy

import (
	"strings"
	"testing"

	"github.com/oskillarrr/traverse/internal/lang"
)

func TestCOBOLFixedFormatAndTransfers(t *testing.T) {
	src := "000100 IDENTIFICATION DIVISION.                                         00010000\n" +
		"000200 PROGRAM-ID.                                                      00020000\n" +
		"000300     COMEN01C.                                                    00030000\n" +
		"       WORKING-STORAGE SECTION.\n" +
		"       01 WS-PGM PIC X(8).\n" +
		"       PROCEDURE DIVISION.\n" +
		"           MOVE 'COADM01C' TO WS-PGM\n" +
		"           MOVE 'COSGN00C' TO WS-PGM\n" +
		"           EXEC CICS XCTL PROGRAM(WS-PGM) END-EXEC.\n" +
		"           EXEC CICS XCTL PROGRAM(CDEMO-TO-PROGRAM) END-EXEC.\n" +
		"           CALL 'CEE3ABD' USING X.\n"
	u := COBOL(lang.Strip(src, lang.COBOL), "x", false)
	if u.Name != "COMEN01C" {
		t.Fatalf("program id = %q (sequence area not ignored)", u.Name)
	}
	var targets []string
	for _, r := range u.Refs {
		if r.Kind == "xctl" {
			targets = append(targets, r.Name)
		}
	}
	if strings.Join(targets, ",") != "COADM01C,COSGN00C" {
		t.Errorf("xctl targets = %v", targets)
	}
	if len(u.Dynamic) != 1 || u.Dynamic[0] != "CDEMO-TO-PROGRAM" {
		t.Errorf("dynamic = %v", u.Dynamic)
	}
	if !IsUtility("CEE3ABD") || !IsUtility("MQPUT") || IsUtility("COSGN00C") {
		t.Errorf("utility classification")
	}
}

func TestJCLDatasets(t *testing.T) {
	src := "//JOB1 JOB (A),CLASS=A\n//S1 EXEC PGM=PROG1\n//IN DD DSN=A.B.C,\n//      DISP=SHR\n//OUT DD DSN=A.B.D(+1),DISP=(NEW,CATLG)\n//TMP DD DSN=&&TEMP,DISP=(NEW,PASS)\n"
	u := JCL(lang.Strip(src, lang.JCL), "x")
	got := map[string]string{}
	for _, r := range u.Refs {
		got[r.Kind+":"+r.Name] = r.Op
	}
	if got["exec:PROG1"] != "" && len(got) != 3 || got["dataset:A.B.C"] != "read" || got["dataset:A.B.D"] != "defines" {
		t.Errorf("refs = %v", got)
	}
	if _, ok := got["dataset:&&TEMP"]; ok {
		t.Errorf("temporary dataset must be ignored")
	}
}
