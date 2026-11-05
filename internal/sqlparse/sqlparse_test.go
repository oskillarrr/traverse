package sqlparse

import "testing"

func TestTables(t *testing.T) {
	cases := []struct {
		sql  string
		want map[string]string
	}{
		{"SELECT a,b FROM ARCHIVO a WHERE a.x=1", map[string]string{"ARCHIVO": "read"}},
		{"select * from CLIENTE c join PEDIDO p on p.cid=c.id", map[string]string{"CLIENTE": "read", "PEDIDO": "read"}},
		{"INSERT INTO EXPEDIENTE (id) VALUES (1)", map[string]string{"EXPEDIENTE": "write"}},
		{"UPDATE FIRMA SET estado='C' WHERE id=?", map[string]string{"FIRMA": "write"}},
		{"DELETE FROM SESION WHERE ts<?", map[string]string{"SESION": "write"}},
		{"SELECT 1 FROM DUAL", map[string]string{}},
		{"just some text, not sql", map[string]string{}},
		{"SELECT * FROM esquema.TABLA", map[string]string{"ESQUEMA.TABLA": "read"}},
	}
	for _, c := range cases {
		got := map[string]string{}
		for _, r := range Tables(c.sql) {
			full, _ := Normalize(r.Table)
			got[full] = r.Op
		}
		if len(got) != len(c.want) {
			t.Errorf("%q -> %v, want %v", c.sql, got, c.want)
			continue
		}
		for k, v := range c.want {
			if got[k] != v {
				t.Errorf("%q: %s=%q want %q", c.sql, k, got[k], v)
			}
		}
	}
}
