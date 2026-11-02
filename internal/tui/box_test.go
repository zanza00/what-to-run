package tui

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestBoxRenderReturnsNilWhenTheSpaceIsTooSmall(t *testing.T) {
	// The box needs one row more than its chrome, whatever the width.
	for _, w := range []int{0, 1, 9, 10, 40} {
		if got := (box{}).render(boxChrome, w); got != nil {
			t.Errorf("render(boxChrome, %d) = %q, want nil", w, got)
		}
	}
	// It needs ten columns, whatever the height.
	for _, h := range []int{0, 1, boxChrome, boxChrome + 1, 24} {
		if got := (box{}).render(h, 9); got != nil {
			t.Errorf("render(%d, 9) = %q, want nil", h, got)
		}
	}
	if got := (box{}).render(boxChrome+1, 10); got == nil {
		t.Error("render(boxChrome+1, 10) = nil, want a box")
	}
}

func TestBoxRenderClampsTheContentWidthToTheFrame(t *testing.T) {
	// The caller asks for more than the frame holds, so the box clamps the
	// content width to w-4 and stays inside the frame.
	got := box{body: []string{"row"}, innerW: 100}.render(24, 20)
	if len(got) == 0 {
		t.Fatal("render(24, 20) = nil, want a box")
	}
	for i, line := range got {
		if w := lipgloss.Width(line); w != 20 {
			t.Errorf("line %d is %d columns wide, want min(innerW, w-4)+4 = 20:\n%q", i, w, line)
		}
	}
}

func TestBoxRenderLines(t *testing.T) {
	// want holds the lines without the ANSI sequences, so the tests pin the
	// glyphs and the widths, not the colors of one theme.
	tests := []struct {
		name string
		b    box
		want []string
	}{
		{
			name: "every part of the drawing",
			b:    box{title: " warn ", body: []string{"0123456789", "ab", ""}, footer: "hi", innerW: 5},
			want: []string{
				"╭─ warn ╮",
				"│ 0123… │",
				"│ ab    │",
				"│       │",
				"│ hi    │",
				"╰───────╯",
			},
		},
		{
			name: "a title wider than the top border is cut without an ellipsis",
			b:    box{title: "this title is way too long", body: []string{"row"}, innerW: 10},
			want: []string{
				"╭─this title ╮",
				"│ row        │",
				"│            │",
				"╰────────────╯",
			},
		},
		{
			name: "a pre-padded body row stays as it is",
			b:    box{body: []string{"ab   "}, innerW: 5},
			want: []string{
				"╭───────╮",
				"│ ab    │",
				"│       │",
				"╰───────╯",
			},
		},
		{
			name: "a footer wider than the content width is dropped",
			b:    box{body: []string{"row"}, footer: "0123456789", innerW: 5},
			want: []string{
				"╭───────╮",
				"│ row   │",
				"│       │",
				"╰───────╯",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.b.render(12, 40)
			if len(got) != len(tt.want) {
				t.Fatalf("render = %d lines, want %d:\n%s", len(got), len(tt.want), strings.Join(got, "\n"))
			}
			for i, line := range got {
				if stripped := ansi.Strip(line); stripped != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, stripped, tt.want[i])
				}
			}
		})
	}
}

func TestBoxRenderReturnsOneLinePerBodyRowPlusChrome(t *testing.T) {
	for _, body := range [][]string{nil, {"one"}, {"one", "two", "three"}} {
		got := box{body: body}.render(24, 40)
		if len(got) != len(body)+boxChrome {
			t.Errorf("render with %d body rows = %d lines, want %d", len(body), len(got), len(body)+boxChrome)
		}
	}
}

func TestBoxRenderBorderColor(t *testing.T) {
	tests := []struct {
		name   string
		border color.Color
	}{
		{"the given color", lipgloss.Color("#FF0000")},
		{"the terminal default", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := box{title: " t ", body: []string{"row"}, footer: "f", border: tt.border, innerW: 10}
			got := b.render(10, 40)
			if len(got) == 0 {
				t.Fatal("render(10, 40) = nil, want a box")
			}
			for i, line := range got {
				if tt.border == nil {
					// The edge style adds no sequence of its own. Only the
					// bold title carries one, on the top border.
					if strings.Contains(line, "38;") {
						t.Errorf("line %d carries a color sequence, want the default foreground:\n%q", i, line)
					}
					if i > 0 && strings.Contains(line, "\x1b") {
						t.Errorf("line %d carries an ANSI sequence, want none:\n%q", i, line)
					}
					continue
				}
				if !strings.Contains(line, "38;2;255;0;0") {
					t.Errorf("line %d does not carry the border color:\n%q", i, line)
				}
			}
		})
	}
}
