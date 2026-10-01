package tableprinter

import (
	"testing"

	"example.com/tool/pkg/iostreams"
)

func TestRender_piped_escapesFields(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	tp := New(ios, "ID", "TEXT")
	tp.AddField("1")
	tp.AddField("line one\nline\ttwo \\ end")
	tp.EndRow()
	if err := tp.Render(); err != nil {
		t.Fatal(err)
	}
	want := "1\tline one\\nline\\ttwo \\\\ end\n"
	if stdout.String() != want {
		t.Errorf("got %q, want %q", stdout.String(), want)
	}
}

func TestRender_tty_alignsByCellWidth(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	ios.SetStdoutTTY(true)
	tp := New(ios, "Name", "N")
	for _, row := range [][2]string{{"日本", "1"}, {"abcd", "2"}} {
		tp.AddField(row[0])
		tp.AddField(row[1])
		tp.EndRow()
	}
	if err := tp.Render(); err != nil {
		t.Fatal(err)
	}
	// "日本" is 2 runes but 4 cells wide, so both rows align to the same column.
	want := "NAME  N\n日本  1\nabcd  2\n"
	if stdout.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", stdout.String(), want)
	}
}
