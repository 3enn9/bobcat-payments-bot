package invoice

import "testing"

func TestEntrepreneurShortName(t *testing.T) {
	if isEntrepreneurName(`ООО "СарСтройТех"`) {
		t.Fatal("ООО should not be entrepreneur")
	}

	cases := map[string]string{
		`ИП Архипов Николай Николаевич`:   "Архипов Н.Н.",
		`ИП Архипов Николай Владимирович`: "Архипов Н.В.",
		`ИП Скрипниченко Иван Петрович`:   "Скрипниченко И.П.",
	}
	for in, want := range cases {
		if !isEntrepreneurName(in) {
			t.Fatalf("expected entrepreneur: %q", in)
		}
		if got := entrepreneurShortName(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestPDFFileName(t *testing.T) {
	cases := []struct {
		number  int
		name    string
		inn     string
		revised bool
		want    string
	}{
		{536, `ООО "СарСтройТех"`, "6454116198", false, "сч 536 сст.pdf"},
		{606, `ИП Архипов Николай Николаевич`, "", true, "сч 606 анн изм.pdf"},
		{12, `ИП Архипов Данила Николаевич`, "", false, "сч 12 адн.pdf"},
	}
	for _, tc := range cases {
		if got := PDFFileName(tc.number, tc.name, tc.inn, tc.revised); got != tc.want {
			t.Fatalf("%q revised=%v: got %q want %q", tc.name, tc.revised, got, tc.want)
		}
	}
}
