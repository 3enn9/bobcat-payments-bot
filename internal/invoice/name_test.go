package invoice

import "testing"

func TestEntrepreneurShortName(t *testing.T) {
	if isEntrepreneurName(`ООО "СарСтройТех"`) {
		t.Fatal("ООО should not be entrepreneur")
	}

	cases := map[string]string{
		`ИП Архипов Николай Николаевич`:   "Архипов Н.Н.",
		`ИП Архипов Николай Владимирович`:  "Архипов Н.В.",
		`ИП Скрипниченко Иван Петрович`:    "Скрипниченко И.П.",
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
