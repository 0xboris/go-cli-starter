package iostreams

// ColorScheme applies basic ANSI styles. Every method is a no-op when color is
// disabled, so call sites never need to branch on TTY/NO_COLOR.
// Color enhances meaning; always pair it with words or icons.
type ColorScheme struct {
	Enabled bool
}

func (c *ColorScheme) wrap(code, s string) string {
	if !c.Enabled || s == "" {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func (c *ColorScheme) Bold(s string) string   { return c.wrap("1", s) }
func (c *ColorScheme) Muted(s string) string  { return c.wrap("90", s) }
func (c *ColorScheme) Red(s string) string    { return c.wrap("31", s) }
func (c *ColorScheme) Green(s string) string  { return c.wrap("32", s) }
func (c *ColorScheme) Yellow(s string) string { return c.wrap("33", s) }
func (c *ColorScheme) Cyan(s string) string   { return c.wrap("36", s) }

func (c *ColorScheme) SuccessIcon() string { return c.Green("✓") }
func (c *ColorScheme) WarningIcon() string { return c.Yellow("!") }
func (c *ColorScheme) FailureIcon() string { return c.Red("X") }

// TableHeader styles table headers (TTY only; piped tables have no header).
func (c *ColorScheme) TableHeader(s string) string { return c.wrap("1;4", s) }
