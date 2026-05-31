package main

import (
	"strings"
	"testing"
)

func TestCompile_LiteralModeMatchesPlainString(t *testing.T) {
	p, err := Compile("foo.bar", "X", true)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	if !p.Match("foo.bar") {
		t.Error("Match() = false; want true for literal dot")
	}
	if p.Match("fooXbar") {
		t.Error("Match() = true; want false")
	}
	if got := p.Replace("foo.bar baz"); got != "X baz" {
		t.Errorf("Replace() = %q; want %q", got, "X baz")
	}
}

func TestCompile_LiteralModeEquivalentToReplaceAll(t *testing.T) {
	cases := []struct {
		find, replace, input string
	}{
		{"alpha", "beta", "alpha alpha"},
		{"(parens)", "[brackets]", "file (parens).txt"},
		{"$dollar", "cent", "cost $dollar"},
		{"a.b", "ab", "a.b.c"},
	}
	for _, tc := range cases {
		p, err := Compile(tc.find, tc.replace, true)
		if err != nil {
			t.Fatalf("Compile(%q): %v", tc.find, err)
		}
		want := strings.ReplaceAll(tc.input, tc.find, tc.replace)
		if got := p.Replace(tc.input); got != want {
			t.Errorf("Replace(%q, %q) on %q = %q; want %q", tc.find, tc.replace, tc.input, got, want)
		}
	}
}

func TestCompile_CaptureGroups(t *testing.T) {
	p, err := Compile(`(\w+)@(\w+)\.(\w+)`, "$2.$1@$3", false)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	got := p.Replace("alice@example.com")
	if got != "example.alice@com" {
		t.Errorf("Replace() = %q; want example.alice@com", got)
	}
}

func TestCompile_NamedCaptureGroup(t *testing.T) {
	p, err := Compile(`(?P<first>\w+) (?P<last>\w+)`, "${last}, ${first}", false)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	got := p.Replace("Ada Lovelace")
	if got != "Lovelace, Ada" {
		t.Errorf("Replace() = %q; want Lovelace, Ada", got)
	}
}

func TestCompile_ReplacementEscapes(t *testing.T) {
	p, err := Compile(`(\d+)`, "$$1 cents", false)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	if got := p.Replace("42"); got != "$1 cents" {
		t.Errorf("Replace() = %q; want %q", got, "$1 cents")
	}
}

func TestCompile_InvalidPatternReturnsError(t *testing.T) {
	_, err := Compile("[unclosed", "x", false)
	if err == nil {
		t.Fatal("Compile() err = nil; want error")
	}
}

func BenchmarkPatternLiteralReplace(b *testing.B) {
	input := strings.Repeat("alpha beta gamma ", 1000)
	p, err := Compile("alpha", "beta", true)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Replace(input)
	}
}

func BenchmarkStringsReplaceAllLiteral(b *testing.B) {
	input := strings.Repeat("alpha beta gamma ", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strings.ReplaceAll(input, "alpha", "beta")
	}
}
