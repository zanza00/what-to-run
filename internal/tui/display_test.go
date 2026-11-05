package tui

import "testing"

// The display tests pin the module seam directly, without the model: the
// start fixes where the frame draws, the five named operations settle the
// facts, and the reads report them. The view tests in tui_test.go pin
// the frames that the model draws through the module, and they stay the
// outermost, byte-level check.

// TestDisplayStart pins what construction derives: a fullscreen start reads
// startedInline() false and draws expanded, and an inline start reads
// startedInline() true and stays anchored while it waits for the size and
// the anchor row.
func TestDisplayStart(t *testing.T) {
	tests := []struct {
		name         string
		inline       bool
		wantStarted  bool
		wantAnchored bool
		wantExpanded bool
	}{
		{"a fullscreen start reads startedInline false", false, false, false, true},
		{"an inline start reads startedInline true", true, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newDisplay(tt.inline)
			if got := d.startedInline(); got != tt.wantStarted {
				t.Errorf("startedInline() = %v, want %v", got, tt.wantStarted)
			}
			if got := d.anchored(); got != tt.wantAnchored {
				t.Errorf("anchored() = %v, want %v", got, tt.wantAnchored)
			}
			if got := d.expanded(); got != tt.wantExpanded {
				t.Errorf("expanded() = %v, want %v", got, tt.wantExpanded)
			}
		})
	}
}

// TestDisplayScreens pins which screen the frame draws on, through the four
// states that the three facts combine into. A fallback keeps the alternate
// screen, so it reads expanded like an expansion does.
func TestDisplayScreens(t *testing.T) {
	tests := []struct {
		name         string
		d            display
		wantAnchored bool
		wantExpanded bool
	}{
		{"a fullscreen start draws expanded", display{}, false, true},
		{"an inline start reads anchored", display{inline: true}, true, false},
		{"an inline start that expanded draws on the alternate screen", display{inline: true, full: true}, false, true},
		{"the fallback stays on the alternate screen", display{inline: true, full: true, noPosition: true}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.anchored(); got != tt.wantAnchored {
				t.Errorf("anchored() = %v, want %v", got, tt.wantAnchored)
			}
			if got := tt.d.expanded(); got != tt.wantExpanded {
				t.Errorf("expanded() = %v, want %v", got, tt.wantExpanded)
			}
		})
	}
}

// TestDisplayOperations pins the five named operations: every settle lands on
// the facts that the old handlers wrote by hand. Each expected display follows
// the old algebra from the starting state, by hand.
func TestDisplayOperations(t *testing.T) {
	tests := []struct {
		name string
		d    display
		op   func(*display)
		want display
	}{
		{
			name: "resize writes the size, marks it ready and remembers the width while anchored",
			d:    display{inline: true},
			op:   func(d *display) { d.resize(80, 30) },
			want: display{inline: true, width: 80, height: 30, ready: true, mainWidth: 80},
		},
		{
			name: "resize keeps the remembered width while the frame draws on the alternate screen",
			d: display{
				inline: true, width: 80, height: 30, ready: true,
				startRow: 5, startRowKnown: true,
			},
			op: func(d *display) { d.expand(); d.resize(85, 30) },
			want: display{
				inline: true, full: true, width: 85, height: 30, ready: true,
				startRow: 5, startRowKnown: true, mainWidth: 80,
			},
		},
		{
			name: "resize on a fullscreen start remembers no width",
			d:    display{},
			op:   func(d *display) { d.resize(80, 30) },
			want: display{width: 80, height: 30, ready: true},
		},
		{
			name: "learnPosition writes the anchor row and clears the fallback",
			d: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
			op: func(d *display) { d.learnPosition(9) },
			want: display{
				inline: true, full: true,
				width: 80, height: 30, ready: true,
				startRow: 9, startRowKnown: true, mainWidth: 80,
			},
		},
		{
			name: "fallBack expands once and remembers the main width",
			d:    display{inline: true, width: 80, height: 30, ready: true},
			op:   func(d *display) { d.fallBack() },
			want: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
		},
		{
			name: "fallBack before the size expands with nothing to remember",
			d:    display{inline: true},
			op:   func(d *display) { d.fallBack() },
			want: display{inline: true, full: true, noPosition: true},
		},
		{
			name: "fallBack after the terminal answered settles nothing",
			d: display{
				inline: true, width: 80, height: 30, ready: true,
				startRow: 5, startRowKnown: true,
			},
			op:   func(d *display) { d.fallBack() },
			want: display{inline: true, width: 80, height: 30, ready: true, startRow: 5, startRowKnown: true},
		},
		{
			name: "fallBack on a fullscreen start settles nothing",
			d:    display{width: 80, height: 30, ready: true},
			op:   func(d *display) { d.fallBack() },
			want: display{width: 80, height: 30, ready: true},
		},
		{
			name: "fallBack settles nothing on a second fire",
			d: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
			op: func(d *display) { d.fallBack() },
			want: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
		},
		{
			name: "fallBack keeps an expansion by the user as it is",
			d: display{
				inline: true, full: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
			op: func(d *display) { d.fallBack() },
			want: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
		},
		{
			name: "expand remembers the main width and goes to the alternate screen",
			d: display{
				inline: true, width: 80, height: 30, ready: true,
				startRow: 5, startRowKnown: true,
			},
			op: func(d *display) { d.expand() },
			want: display{
				inline: true, full: true, width: 80, height: 30, ready: true,
				startRow: 5, startRowKnown: true, mainWidth: 80,
			},
		},
		{
			name: "expand on the alternate screen settles nothing",
			d: display{
				inline: true, full: true,
				width: 85, height: 30, ready: true, mainWidth: 80,
			},
			op:   func(d *display) { d.expand() },
			want: display{inline: true, full: true, width: 85, height: 30, ready: true, mainWidth: 80},
		},
		{
			name: "expand under the fallback goes nowhere",
			d: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
			op: func(d *display) { d.expand() },
			want: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
		},
		{
			name: "expand on a fullscreen start goes nowhere",
			d:    display{width: 80, height: 30, ready: true},
			op:   func(d *display) { d.expand() },
			want: display{width: 80, height: 30, ready: true},
		},
		{
			name: "collapse returns to the area and keeps the remembered width",
			d: display{
				inline: true, full: true,
				width: 85, height: 30, ready: true,
				startRow: 5, startRowKnown: true, mainWidth: 80,
			},
			op: func(d *display) { d.collapse() },
			want: display{
				inline: true, width: 85, height: 30, ready: true,
				startRow: 5, startRowKnown: true, mainWidth: 80,
			},
		},
		{
			name: "collapse under the fallback never returns",
			d: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
			op: func(d *display) { d.collapse() },
			want: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true, mainWidth: 80,
			},
		},
		{
			name: "collapse on an anchored display settles nothing",
			d: display{
				inline: true, width: 80, height: 30, ready: true,
				startRow: 5, startRowKnown: true, mainWidth: 80,
			},
			op: func(d *display) { d.collapse() },
			want: display{
				inline: true, width: 80, height: 30, ready: true,
				startRow: 5, startRowKnown: true, mainWidth: 80,
			},
		},
		{
			name: "collapse on a fullscreen start goes nowhere",
			d:    display{width: 80, height: 30, ready: true},
			op:   func(d *display) { d.collapse() },
			want: display{width: 80, height: 30, ready: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.d
			tt.op(&d)
			if d != tt.want {
				t.Errorf("after the operation the display holds %+v, want %+v", d, tt.want)
			}
		})
	}
}

// TestDisplayRefusals pins the reports of the three operations that can
// refuse: the key handlers settle a list window only when the report says
// that the frame moved.
func TestDisplayRefusals(t *testing.T) {
	tests := []struct {
		name string
		d    display
		op   func(*display) bool
		want bool
	}{
		{
			name: "fallBack reports the move",
			d:    display{inline: true, width: 80, height: 30, ready: true},
			op:   func(d *display) bool { return d.fallBack() },
			want: true,
		},
		{
			name: "fallBack reports no move after the terminal answered",
			d:    display{inline: true, startRow: 3, startRowKnown: true},
			op:   func(d *display) bool { return d.fallBack() },
			want: false,
		},
		{
			name: "fallBack reports no move on a fullscreen start",
			d:    display{},
			op:   func(d *display) bool { return d.fallBack() },
			want: false,
		},
		{
			name: "fallBack reports no move on a second fire",
			d: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true,
			},
			op:   func(d *display) bool { return d.fallBack() },
			want: false,
		},
		{
			name: "expand reports the move out of the area",
			d: display{
				inline: true, width: 80, height: 30, ready: true,
				startRow: 5, startRowKnown: true,
			},
			op:   func(d *display) bool { return d.expand() },
			want: true,
		},
		{
			name: "expand reports no move under the fallback",
			d: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true,
			},
			op:   func(d *display) bool { return d.expand() },
			want: false,
		},
		{
			name: "expand reports no move on a fullscreen start",
			d:    display{width: 80, height: 30, ready: true},
			op:   func(d *display) bool { return d.expand() },
			want: false,
		},
		{
			name: "collapse reports the move back to the area",
			d: display{
				inline: true, full: true,
				width: 85, height: 30, ready: true,
				startRow: 5, startRowKnown: true, mainWidth: 80,
			},
			op:   func(d *display) bool { return d.collapse() },
			want: true,
		},
		{
			name: "collapse reports no move under the fallback",
			d: display{
				inline: true, full: true, noPosition: true,
				width: 80, height: 30, ready: true,
			},
			op:   func(d *display) bool { return d.collapse() },
			want: false,
		},
		{
			name: "collapse reports no move on a fullscreen start",
			d:    display{width: 80, height: 30, ready: true},
			op:   func(d *display) bool { return d.collapse() },
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.d
			if got := tt.op(&d); got != tt.want {
				t.Errorf("the operation reports %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDisplayReads pins the derived facts: the area height waits for the size
// and the anchor row and never drops below the minimum, the inline width
// keeps the remembered width of the main screen after a collapse on a wider
// alternate screen, and the view height is the natural size before the
// terminal reports its size. naturalRows is the natural height of the list.
func TestDisplayReads(t *testing.T) {
	t.Run("areaHeight", func(t *testing.T) {
		tests := []struct {
			name string
			d    display
			want int
		}{
			{"0 before the size", display{inline: true, startRow: 5, startRowKnown: true}, 0},
			{"0 before the anchor row", display{inline: true, width: 80, height: 30, ready: true}, 0},
			{"the free space below the prompt", display{
				inline: true, width: 80, height: 30, ready: true,
				startRow: 10, startRowKnown: true,
			}, 20},
			{"the minimum floor when the terminal is full", display{
				inline: true, width: 80, height: 17, ready: true,
				startRow: 15, startRowKnown: true,
			}, minInlineHeight},
			{"exactly the floor", display{
				inline: true, width: 80, height: 30, ready: true,
				startRow: 15, startRowKnown: true,
			}, 15},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.d.areaHeight(); got != tt.want {
					t.Errorf("areaHeight() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("inlineWidth", func(t *testing.T) {
		tests := []struct {
			name string
			d    display
			want int
		}{
			{"the raw width before anything is remembered", display{inline: true, width: 40}, 40},
			{"the remembered main width after a collapse on a wider alternate screen", display{
				inline: true, width: 85, mainWidth: 80,
			}, 80},
			{"the remembered main width caps a narrower terminal", display{
				inline: true, width: 60, mainWidth: 80,
			}, 60},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.d.inlineWidth(); got != tt.want {
					t.Errorf("inlineWidth() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("viewHeight", func(t *testing.T) {
		tests := []struct {
			name        string
			d           display
			naturalRows int
			want        int
		}{
			{"the natural size before ready", display{width: 80, height: 24}, 7, 11},
			{"the terminal height once ready", display{width: 80, height: 24, ready: true}, 7, 24},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.d.viewHeight(tt.naturalRows); got != tt.want {
					t.Errorf("viewHeight(%d) = %d, want %d", tt.naturalRows, got, tt.want)
				}
			})
		}
	})
}
