package cmdutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"example.com/tool/pkg/iostreams"
)

type pflagFlag = pflag.Flag

// Exporter writes --json output, honoring --jq and --template.
// It is nil when --json was not passed: commands branch on `opts.Exporter != nil`.
type Exporter interface {
	Fields() []string
	Write(ios *iostreams.IOStreams, data any) error
}

// exportable is implemented by models to return only the requested fields.
type exportable interface {
	ExportData(fields []string) map[string]any
}

// AddJSONFlags adds --json, --jq and --template to cmd. Requested fields are
// validated against fields; a bare --json lists them.
func AddJSONFlags(cmd *cobra.Command, exportTarget *Exporter, fields []string) {
	f := cmd.Flags()
	f.StringSlice("json", nil, "Output JSON with the specified `fields`")
	f.StringP("jq", "q", "", "Filter JSON output using a jq `expression`")
	f.StringP("template", "t", "", "Format JSON output using a Go template")

	sorted := append([]string(nil), fields...)
	sort.Strings(sorted)
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["help:json-fields"] = strings.Join(sorted, ",")

	_ = cmd.RegisterFlagCompletionFunc("json", func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		prefix := ""
		if i := strings.LastIndex(toComplete, ","); i >= 0 {
			prefix = toComplete[:i+1]
		}
		var out []string
		for _, f := range sorted {
			out = append(out, prefix+f)
		}
		return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	})

	// A bare `--json` lists the available fields instead of a generic parse error.
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if c == cmd && err.Error() == "flag needs an argument: --json" {
			return FlagErrorf("Specify one or more comma-separated fields for `--json`:\n  %s", strings.Join(sorted, "\n  "))
		}
		if c.HasParent() {
			return c.Parent().FlagErrorFunc()(c, err)
		}
		return FlagErrorWrap(err)
	})

	oldPreRun := cmd.PreRunE
	cmd.PreRunE = func(c *cobra.Command, args []string) error {
		if oldPreRun != nil {
			if err := oldPreRun(c, args); err != nil {
				return err
			}
		}
		requested, _ := f.GetStringSlice("json")
		jqExpr, _ := f.GetString("jq")
		tmpl, _ := f.GetString("template")
		if len(requested) == 0 {
			if jqExpr != "" {
				return FlagErrorf("cannot use `--jq` without specifying `--json`")
			}
			if tmpl != "" {
				return FlagErrorf("cannot use `--template` without specifying `--json`")
			}
			return nil
		}
		if jqExpr != "" && tmpl != "" {
			return FlagErrorf("cannot use `--jq` and `--template` together")
		}
		for _, r := range requested {
			if !contains(fields, r) {
				return FlagErrorf("Unknown JSON field: %q\nAvailable fields:\n  %s", r, strings.Join(sorted, "\n  "))
			}
		}
		*exportTarget = &jsonExporter{fields: requested, jq: jqExpr, template: tmpl}
		return nil
	}
}

type jsonExporter struct {
	fields   []string
	jq       string
	template string
}

func (e *jsonExporter) Fields() []string { return e.fields }

func (e *jsonExporter) Write(ios *iostreams.IOStreams, data any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep URLs readable
	if ios.IsStdoutTTY() && e.jq == "" && e.template == "" {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(e.exportData(reflect.ValueOf(data))); err != nil {
		return err
	}

	switch {
	case e.jq != "":
		return evalJQ(&buf, ios.Out, e.jq)
	case e.template != "":
		var v any
		if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
			return err
		}
		t, err := template.New("").Funcs(template.FuncMap{
			"json": func(v any) (string, error) { b, err := json.Marshal(v); return string(b), err },
			"join": func(sep string, v []any) string {
				s := make([]string, len(v))
				for i := range v {
					s[i] = fmt.Sprint(v[i])
				}
				return strings.Join(s, sep)
			},
		}).Parse(e.template)
		if err != nil {
			return FlagErrorf("invalid template: %v", err)
		}
		return t.Execute(ios.Out, v)
	default:
		_, err := io.Copy(ios.Out, &buf)
		return err
	}
}

// exportData walks slices/pointers and calls ExportData where available.
// An empty slice exports as [] (never null).
func (e *jsonExporter) exportData(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return e.exportData(v.Elem())
	case reflect.Slice:
		out := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = e.exportData(v.Index(i))
		}
		return out
	}
	if v.CanInterface() {
		if ex, ok := v.Interface().(exportable); ok {
			return ex.ExportData(e.fields)
		}
		return v.Interface()
	}
	return nil
}

func evalJQ(in io.Reader, out io.Writer, expr string) error {
	query, err := gojq.Parse(expr)
	if err != nil {
		return FlagErrorf("invalid jq expression: %v", err)
	}
	var input any
	if err := json.NewDecoder(in).Decode(&input); err != nil {
		return err
	}
	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			return nil
		}
		if err, ok := v.(error); ok {
			return err
		}
		if s, ok := v.(string); ok { // raw strings, like `jq -r`
			fmt.Fprintln(out, s)
			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(b))
	}
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
