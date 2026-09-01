package invoice

import "testing"

func TestSupplierVATRate(t *testing.T) {
	cases := []struct {
		name string
		inn  string
		want int
	}{
		{`ООО "СарСтройТех"`, "6454116198", 22},
		{`ИП Архипов Данила Николаевич`, "", 5},
		{`ИП Архипов Николай Николаевич`, "", 0},
		{`ИП Скрипниченко Иван Петрович`, "", 0},
	}
	for _, tc := range cases {
		if got := SupplierVATRate(tc.name, tc.inn); got != tc.want {
			t.Fatalf("%q: got %d want %d", tc.name, got, tc.want)
		}
	}
}

func TestCalcVAT(t *testing.T) {
	if got := CalcVAT(12000, `ООО "СарСтройТех"`, "6454116198"); got != 2163.93 {
		t.Fatalf("22%% VAT: got %v want 2163.93", got)
	}
	if got := CalcVAT(10500, `ИП Архипов Данила Николаевич`, ""); got != 500 {
		t.Fatalf("5%% VAT: got %v want 500", got)
	}
	if got := CalcVAT(10000, `ИП Скрипниченко`, ""); got != 0 {
		t.Fatalf("no VAT: got %v want 0", got)
	}
}
