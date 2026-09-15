package ui

import (
	"reflect"
	"testing"
)

func TestParseLang(t *testing.T) {
	tests := []struct {
		in   string
		want Lang
		ok   bool
	}{
		{"", LangEN, true}, // the zero value has to land on the default
		{"en", LangEN, true},
		{"EN", LangEN, true},
		{"english", LangEN, true},
		{"ja", LangJA, true},
		{"JP", LangJA, true},
		{" ja ", LangJA, true},
		{"日本語", LangJA, true},
		{"fr", LangEN, false}, // unrecognized: reported, but still English
	}
	for _, tt := range tests {
		got, ok := ParseLang(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ParseLang(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestDefaultLangIsEnglish(t *testing.T) {
	// The zero Config must produce an English interface: prpr is published
	// where most readers do not read Japanese.
	var zero Lang
	if got := Catalog(zero); got.TabAll != english.TabAll {
		t.Errorf("the zero Lang produced %q for the first tab, want the English %q", got.TabAll, english.TabAll)
	}
}

func TestEveryStringIsTranslated(t *testing.T) {
	// A missing translation would otherwise show up as a blank label at
	// runtime, on whichever screen happens to use it.
	for _, c := range []struct {
		name string
		s    Strings
	}{{"english", english}, {"japanese", japanese}} {
		v := reflect.ValueOf(c.s)
		for i := range v.NumField() {
			if v.Field(i).String() == "" {
				t.Errorf("%s catalog: %s is empty", c.name, v.Type().Field(i).Name)
			}
		}
	}
}
