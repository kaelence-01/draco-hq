package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// Basic ANSI colors follow the terminal's own palette, light or dark.
// applyTheme sets them; screens read them when they draw.
var (
	accentStyle lipgloss.Style
	boldStyle   lipgloss.Style
	faintStyle  lipgloss.Style
	warnStyle   lipgloss.Style
	okStyle     lipgloss.Style
	errStyle    lipgloss.Style
	logoStyle   lipgloss.Style
	ruleStyle   lipgloss.Style
	boxStyle    lipgloss.Style
	helpStyles  help.Styles
	formTheme   huh.Theme
	theme       string // the one applied
)

// defaultTheme is the theme without one in the config.
const defaultTheme = "ansi"

// themes are the built-in themes, in the order Configuración lists them.
var themes = []struct{ name, desc string }{
	{"ansi", "Los colores de tu terminal (por defecto)"},
	{"alto-contraste", "Colores brillantes y negrita en lo importante"},
	{"mono", "Sin colores: negrita, tenue y subrayado"},
}

func init() { applyTheme(defaultTheme) }

// knownTheme is name when it is a built-in theme, else the default.
func knownTheme(name string) string {
	for _, t := range themes {
		if t.name == name {
			return name
		}
	}
	return defaultTheme
}

// applyTheme sets the styles of a theme; an unknown name is the default.
func applyTheme(name string) {
	theme = knownTheme(name)
	plain := lipgloss.NewStyle()
	box := plain.Border(lipgloss.RoundedBorder()).Padding(0, 1)
	boldStyle, faintStyle = plain.Bold(true), plain.Faint(true)
	switch theme {
	case "alto-contraste":
		accentStyle = plain.Foreground(lipgloss.BrightCyan).Bold(true)
		warnStyle = plain.Foreground(lipgloss.BrightYellow).Bold(true)
		okStyle = plain.Foreground(lipgloss.BrightGreen)
		errStyle = plain.Foreground(lipgloss.BrightRed).Bold(true)
		logoStyle = accentStyle
		faintStyle = plain.Foreground(lipgloss.White)
		ruleStyle = plain.Foreground(lipgloss.White)
		boxStyle = box.BorderForeground(lipgloss.White)
		helpStyles = help.Styles{ShortKey: plain.Foreground(lipgloss.BrightWhite).Bold(true), ShortDesc: plain.Foreground(lipgloss.White),
			ShortSeparator: plain.Foreground(lipgloss.White), Ellipsis: plain.Foreground(lipgloss.White),
			FullKey: plain.Foreground(lipgloss.BrightWhite).Bold(true), FullDesc: plain.Foreground(lipgloss.White), FullSeparator: plain.Foreground(lipgloss.White)}
		formTheme = huh.ThemeFunc(huh.ThemeBase16)
	case "mono":
		accentStyle = plain.Bold(true)
		warnStyle = plain.Bold(true)
		okStyle = plain
		errStyle = plain.Bold(true).Underline(true)
		logoStyle = plain.Bold(true)
		ruleStyle = plain.Faint(true)
		boxStyle = box
		helpStyles = help.Styles{ShortKey: plain.Bold(true), ShortDesc: plain, ShortSeparator: plain.Faint(true), Ellipsis: plain.Faint(true),
			FullKey: plain.Bold(true), FullDesc: plain, FullSeparator: plain.Faint(true)}
		formTheme = huh.ThemeFunc(monoForm)
	default:
		accentStyle = plain.Foreground(lipgloss.Cyan)
		warnStyle = plain.Foreground(lipgloss.Yellow)
		okStyle = plain.Foreground(lipgloss.Green)
		errStyle = plain.Foreground(lipgloss.Red)
		logoStyle = plain.Foreground(lipgloss.Cyan).Bold(true)
		ruleStyle = plain.Foreground(lipgloss.BrightBlack)
		boxStyle = box.BorderForeground(lipgloss.BrightBlack)
		helpStyles = help.New().Styles
		formTheme = huh.ThemeFunc(huh.ThemeBase16)
	}
}

// monoForm is huh's base theme without its few colors.
func monoForm(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)
	button := lipgloss.NewStyle().Padding(0, 2).MarginRight(1)
	for _, f := range []*huh.FieldStyles{&t.Focused, &t.Blurred} {
		f.FocusedButton = button.Reverse(true)
		f.BlurredButton = button
		f.TextInput.Placeholder = lipgloss.NewStyle().Faint(true)
	}
	t.Help = helpStyles
	return t
}
