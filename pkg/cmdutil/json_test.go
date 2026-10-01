package cmdutil

import (
	"testing"

	"example.com/tool/internal/api"
	"example.com/tool/pkg/iostreams"
)

func TestJSONExporter(t *testing.T) {
	items := []api.Item{{ID: 1, Title: "a <b>", State: "open"}}
	tests := []struct {
		name string
		tty  bool
		e    jsonExporter
		want string
	}{
		{name: "piped is compact, selected fields only, no HTML escaping", e: jsonExporter{fields: []string{"id", "title"}},
			want: `[{"id":1,"title":"a <b>"}]` + "\n"},
		{name: "tty is indented", tty: true, e: jsonExporter{fields: []string{"id"}},
			want: "[\n  {\n    \"id\": 1\n  }\n]\n"},
		{name: "jq prints raw strings", e: jsonExporter{fields: []string{"title"}, jq: ".[].title"},
			want: "a <b>\n"},
		{name: "template", e: jsonExporter{fields: []string{"id", "state"}, template: `{{range .}}{{.id}}={{.state}}{{end}}`},
			want: "1=open"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, _ := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			if err := tt.e.Write(ios, items); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != tt.want {
				t.Errorf("got %q, want %q", stdout.String(), tt.want)
			}
		})
	}
}

func TestJSONExporter_emptyIsArray(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	e := jsonExporter{fields: []string{"id"}}
	if err := e.Write(ios, []api.Item{}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "[]\n" {
		t.Errorf("got %q, want []", stdout.String())
	}
}
