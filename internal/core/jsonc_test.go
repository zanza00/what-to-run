package core

import (
	"reflect"
	"testing"
)

func TestUnmarshalJSONC(t *testing.T) {
	type target struct {
		Name   string         `json:"name"`
		Tags   []string       `json:"tags"`
		Nested map[string]int `json:"nested"`
	}

	tests := []struct {
		name    string
		data    string
		want    target
		wantErr bool
	}{
		{
			name: "standard json decodes",
			data: `{"name": "plain", "tags": ["a", "b"], "nested": {"x": 1}}`,
			want: target{Name: "plain", Tags: []string{"a", "b"}, Nested: map[string]int{"x": 1}},
		},
		{
			name: "line comments",
			data: `{
				// the name
				"name": "line",
				"tags": ["a"], // an inline comment
				"nested": {}
			}`,
			want: target{Name: "line", Tags: []string{"a"}, Nested: map[string]int{}},
		},
		{
			name: "block comments",
			data: `{
				/* the name */
				"name": "block",
				/*
					a multi-line comment
				*/
				"tags": ["a"],
				"nested": {}
			}`,
			want: target{Name: "block", Tags: []string{"a"}, Nested: map[string]int{}},
		},
		{
			name: "trailing commas",
			data: `{"name": "trailing", "tags": ["a", "b",], "nested": {"x": 2,},}`,
			want: target{Name: "trailing", Tags: []string{"a", "b"}, Nested: map[string]int{"x": 2}},
		},
		{
			name: "comment markers inside strings are preserved",
			data: `{"name": "a // b /* c */ d", "tags": ["// not a comment", "/* neither */"], "nested": {}}`,
			want: target{Name: "a // b /* c */ d", Tags: []string{"// not a comment", "/* neither */"}, Nested: map[string]int{}},
		},
		{
			name: "escaped quotes inside strings",
			data: `{"name": "say \"hi\" now", "tags": ["tail \\", "quote \""], "nested": {}}`,
			want: target{Name: `say "hi" now`, Tags: []string{`tail \`, `quote "`}, Nested: map[string]int{}},
		},
		{
			name:    "truncated json",
			data:    `{"name": "broken",`,
			wantErr: true,
		},
		{
			name:    "malformed json",
			data:    `{"name": }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got target
			err := unmarshalJSONC([]byte(tt.data), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("unmarshalJSONC() = %+v, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unmarshalJSONC() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unmarshalJSONC() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
