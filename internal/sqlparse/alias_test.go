package sqlparse

import "testing"

func TestJPQLNavigationIsNotATable(t *testing.T) {
	got := Tables("select r from Usuario u join u.roles r where u.id = :id")
	if len(got) != 1 || got[0].Table != "Usuario" {
		t.Fatalf("got %+v", got)
	}
	got = Tables("SELECT a.ID FROM AVISOS a JOIN a.roles x")
	if len(got) != 1 || got[0].Table != "AVISOS" {
		t.Fatalf("got %+v", got)
	}
}

func TestCommaJoinsAndSchemas(t *testing.T) {
	got := Tables("SELECT p.NOMBRE, u.NOMBRE FROM ORDERS.PERSONAS p, AUDIT.ORDERS_PROCESS s, UNIDADES u WHERE p.ID = u.ID")
	want := map[string]bool{"ORDERS.PERSONAS": true, "AUDIT.ORDERS_PROCESS": true, "UNIDADES": true}
	if len(got) != 3 {
		t.Fatalf("got %+v", got)
	}
	for _, r := range got {
		if !want[r.Table] {
			t.Errorf("unexpected %s", r.Table)
		}
	}
}
