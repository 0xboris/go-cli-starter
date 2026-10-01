// Package tableprinter renders aligned, colored tables on a TTY and plain
// tab-separated values (no header, no truncation, no color) when piped.
// Piped fields escape \\, tab, CR and LF so each record stays on one line.
// Widths are measured in terminal cells (CJK and emoji take two).
// For production, consider github.com/cli/go-gh/v2/pkg/tableprinter.
package tableprinter

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"

	"example.com/tool/internal/text"
	"example.com/tool/pkg/iostreams"
)

type field struct {
	text  string
	color func(string) string
}

type TablePrinter struct {
	io     *iostreams.IOStreams
	isTTY  bool
	header []string
	rows   [][]field
	cur    []field
}

func New(ios *iostreams.IOStreams, header ...string) *TablePrinter {
	return &TablePrinter{io: ios, isTTY: ios.IsStdoutTTY(), header: header}
}

func (t *TablePrinter) AddField(s string, color ...func(string) string) {
	f := field{text: s}
	if len(color) > 0 {
		f.color = color[0]
	}
	t.cur = append(t.cur, f)
}

// AddTimeField renders relative time on a TTY and RFC3339 when piped.
func (t *TablePrinter) AddTimeField(now, ts time.Time, color ...func(string) string) {
	if t.isTTY {
		t.AddField(text.FuzzyAgo(now, ts), color...)
		return
	}
	t.AddField(ts.UTC().Format(time.RFC3339))
}

func (t *TablePrinter) EndRow() {
	t.rows = append(t.rows, t.cur)
	t.cur = nil
}

func (t *TablePrinter) Render() error {
	out := t.io.Out
	if !t.isTTY {
		for _, r := range t.rows {
			cells := make([]string, len(r))
			for i, f := range r {
				cells[i] = escapeField(f.text)
			}
			if _, err := fmt.Fprintln(out, strings.Join(cells, "\t")); err != nil {
				return err
			}
		}
		return nil
	}

	cs := t.io.ColorScheme()
	all := t.rows
	if len(t.header) > 0 {
		h := make([]field, len(t.header))
		for i, s := range t.header {
			h[i] = field{text: strings.ToUpper(s), color: cs.TableHeader}
		}
		all = append([][]field{h}, all...)
	}
	widths := map[int]int{}
	for _, r := range all {
		for i, f := range r {
			if n := runewidth.StringWidth(f.text); n > widths[i] {
				widths[i] = n
			}
		}
	}
	maxWidth := t.io.TerminalWidth()
	for _, r := range all {
		var b strings.Builder
		used := 0
		for i, f := range r {
			s := f.text
			last := i == len(r)-1
			if last { // truncate only the last column to the terminal width
				if room := maxWidth - used; room > 3 && runewidth.StringWidth(s) > room {
					s = runewidth.Truncate(s, room, "...")
				}
			}
			pad := ""
			if !last {
				pad = strings.Repeat(" ", widths[i]-runewidth.StringWidth(s)+2)
			}
			if f.color != nil {
				s = f.color(s)
			}
			b.WriteString(s + pad)
			used += widths[i] + 2
		}
		if _, err := fmt.Fprintln(out, strings.TrimRight(b.String(), " ")); err != nil {
			return err
		}
	}
	return nil
}

var fieldEscaper = strings.NewReplacer("\\", "\\\\", "\t", "\\t", "\r", "\\r", "\n", "\\n")

// escapeField makes a value safe for one tab-separated record.
func escapeField(s string) string { return fieldEscaper.Replace(s) }
