package main

import (
	"path/filepath"
	"testing"
)

func TestProgName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/usr/local/bin/wtr", "wtr"},
		{"what-to-run", "what-to-run"},
		{"bin/wtr", "wtr"},
		{"", "wtr"},
	}
	for _, tt := range tests {
		if got := progName(tt.in); got != tt.want {
			t.Errorf("progName(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestVersionLine(t *testing.T) {
	tests := []struct {
		prog    string
		version string
		want    string
	}{
		{"wtr", "0.1.0", "wtr 0.1.0"},
		{"what-to-run", "dev", "what-to-run dev"},
	}
	for _, tt := range tests {
		if got := versionLine(tt.prog, tt.version); got != tt.want {
			t.Errorf("versionLine(%q, %q) = %q; want %q", tt.prog, tt.version, got, tt.want)
		}
	}
}

func TestWidgetBin(t *testing.T) {
	// A bare name is kept: the widget resolves it through PATH when the key
	// is pressed, so it keeps calling the binary by the invoked name.
	if got := widgetBin("what-to-run"); got != "what-to-run" {
		t.Errorf("widgetBin(%q) = %q; want the bare name", "what-to-run", got)
	}

	// A path is made absolute, so the widget runs exactly that copy.
	abs, err := filepath.Abs("bin/wtr")
	if err != nil {
		t.Fatal(err)
	}
	if got := widgetBin("bin/wtr"); got != abs {
		t.Errorf("widgetBin(%q) = %q; want %q", "bin/wtr", got, abs)
	}

	// An absolute path is kept as is.
	if got := widgetBin(abs); got != abs {
		t.Errorf("widgetBin(%q) = %q; want the same path", abs, got)
	}
}

func TestHelpHint(t *testing.T) {
	const wantHint = "The same program is also available as 'wtr', quicker to type."
	tests := []struct {
		name  string
		alias bool
		want  string
	}{
		{"what-to-run", true, wantHint},
		{"what-to-run", false, ""},
		{"wtr", true, ""},
		{"", true, ""},
	}
	for _, tt := range tests {
		if got := helpHint(tt.name, tt.alias); got != tt.want {
			t.Errorf("helpHint(%q, %v) = %q; want %q", tt.name, tt.alias, got, tt.want)
		}
	}
}
