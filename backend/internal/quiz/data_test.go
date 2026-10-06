package quiz

import (
	"strings"
	"testing"
)

const valid = `{"questions":[{"id":"q1","text":"T","options":[{"key":"A","text":"a"},{"key":"B","text":"b"}],"points":10,"defaultDurationSec":30}]}`

func paths(errs []FieldError) string {
	var p []string
	for _, e := range errs {
		p = append(p, e.Path)
	}
	return strings.Join(p, ",")
}

func TestValid(t *testing.T) {
	s, errs := ParseBytes([]byte(valid))
	if errs != nil || len(s.Questions) != 1 || *s.Questions[0].DefaultDurationSec != 30 {
		t.Fatalf("unexpected: %v %+v", errs, s)
	}
}

func TestInvalid(t *testing.T) {
	opt := func(n int) string {
		var o []string
		for i := 0; i < n; i++ {
			o = append(o, `{"key":"`+string(rune('A'+i))+`","text":"x"}`)
		}
		return "[" + strings.Join(o, ",") + "]"
	}
	cases := []struct{ name, in, want string }{
		{"not json", `nope`, "$"},
		{"no questions", `{}`, "questions"},
		{"empty", `{"questions":[]}`, "questions"},
		{"missing id", `{"questions":[{"text":"t","options":` + opt(2) + `}]}`, "questions[0].id"},
		{"dup ids", `{"questions":[{"id":"a","text":"t","options":` + opt(2) + `},{"id":"a","text":"t","options":` + opt(2) + `}]}`, "questions[1].id"},
		{"1 option", `{"questions":[{"id":"a","text":"t","options":` + opt(1) + `}]}`, "questions[0].options"},
		{"7 options", `{"questions":[{"id":"a","text":"t","options":` + opt(7) + `}]}`, "questions[0].options"},
		{"dup key", `{"questions":[{"id":"a","text":"t","options":[{"key":"A","text":"x"},{"key":"A","text":"y"}]}]}`, "questions[0].options[1].key"},
		{"bad duration", `{"questions":[{"id":"a","text":"t","options":` + opt(2) + `,"defaultDurationSec":20}]}`, "questions[0].defaultDurationSec"},
		{"neg points", `{"questions":[{"id":"a","text":"t","options":` + opt(2) + `,"points":-1}]}`, "questions[0].points"},
		{"multiple", `{"questions":[{"text":"t","options":` + opt(1) + `}]}`, "questions[0].id,questions[0].options"},
	}
	for _, c := range cases {
		_, errs := ParseBytes([]byte(c.in))
		if errs == nil || paths(errs) != c.want {
			t.Errorf("%s: got %q want %q", c.name, paths(errs), c.want)
		}
	}
}

func TestCorrectKeyMustMatchAnOption(t *testing.T) {
	ok := `{"questions":[{"id":"q1","text":"x","options":[{"key":"A","text":"a"},{"key":"B","text":"b"}],"correctKey":"B"}]}`
	set, errs := ParseBytes([]byte(ok))
	if errs != nil || set.Questions[0].CorrectKey != "B" {
		t.Fatalf("valid correctKey rejected: %v", errs)
	}
	bad := `{"questions":[{"id":"q1","text":"x","options":[{"key":"A","text":"a"},{"key":"B","text":"b"}],"correctKey":"Z"}]}`
	if _, errs := ParseBytes([]byte(bad)); len(errs) != 1 || errs[0].Path != "questions[0].correctKey" {
		t.Fatalf("expected correctKey error, got %v", errs)
	}
}
