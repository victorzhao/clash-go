package app

import "github.com/egoist/mygo/ui"

// The app's colors, in the manner of FlClash's dashboard: a gray page,
// white cards with a hairline edge, and a sidebar a shade darker. The dark
// theme follows a navy shield palette, with a green accent for the state
// that everything works.
var (
	lightPage    = ui.Hex("#F4F5F7")
	lightCard    = ui.Hex("#FFFFFF")
	lightSidebar = ui.Hex("#ECEEF2")
	darkPage     = ui.Hex("#0F172A")
	darkCard     = ui.Hex("#192134")
	darkSidebar  = ui.Hex("#0C1424")
)

// applyTheme sets the frame's theme from the settings, with the tokens the
// interface uses for its cards, text and accents.
func (a *App) applyTheme(c *ui.Context) {
	var base *ui.Theme
	switch a.settings.Theme {
	case "light":
		base = ui.LightTheme()
	case "dark":
		base = ui.DarkTheme()
	default:
		base = ui.LightTheme()
		if c.Theme().Dark {
			base = ui.DarkTheme()
		}
	}
	if base.Dark {
		base.Background = darkPage
		base.Surface = ui.Hex("#202B44")
		base.SurfaceHover = ui.Hex("#283554")
		base.SurfacePressed = ui.Hex("#314066")
		base.Border = ui.RGBA(255, 255, 255, 0.09)
		base.Text = ui.Hex("#F8FAFC")
		base.TextMuted = ui.Hex("#94A3B8")
		base.Accent = ui.Hex("#22C55E")
		base.AccentHover = ui.Hex("#34D399")
		base.AccentPressed = ui.Hex("#16A34A")
		base.AccentText = ui.Hex("#0B1220")
		base.Danger = ui.Hex("#F87171")
		base.Warning = ui.Hex("#FBBF24")
		base.Success = ui.Hex("#22C55E")
		base.Selection = ui.RGBA(34, 197, 94, 0.3)
		base.Focus = ui.RGBA(34, 197, 94, 0.6)
		base.Scrollbar = ui.RGBA(255, 255, 255, 0.18)
	} else {
		base.Background = lightPage
		base.Surface = ui.Hex("#FFFFFF")
		base.SurfaceHover = ui.Hex("#F3F4F6")
		base.SurfacePressed = ui.Hex("#E8EAEE")
		base.Border = ui.RGBA(15, 23, 42, 0.10)
		base.Text = ui.Hex("#0F172A")
		base.TextMuted = ui.Hex("#64748B")
		base.Accent = ui.Hex("#2563EB")
		base.AccentHover = ui.Hex("#3B82F6")
		base.AccentPressed = ui.Hex("#1D4ED8")
		base.AccentText = ui.Hex("#FFFFFF")
		base.Danger = ui.Hex("#DC2626")
		base.Warning = ui.Hex("#D97706")
		base.Success = ui.Hex("#16A34A")
		base.Selection = ui.RGBA(37, 99, 235, 0.22)
		base.Focus = ui.RGBA(37, 99, 235, 0.55)
		base.Scrollbar = ui.RGBA(15, 23, 42, 0.22)
	}
	base.Radius = 8
	base.Spacing = 4
	c.SetTheme(base)
}

// cardHoverColor is the background of a card under the pointer.
func cardHoverColor(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.Hex("#1D2740")
	}
	return ui.Hex("#FAFBFC")
}

// navActiveColor is the background of the sidebar item shown.
func navActiveColor(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.RGBA(34, 197, 94, 0.16)
	}
	return ui.RGBA(15, 23, 42, 0.07)
}

// cardColor is the color of a card: white in the light theme, a raised
// navy in the dark one.
func cardColor(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return darkCard
	}
	return lightCard
}

// sidebarColor is the color of the sidebar.
func sidebarColor(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return darkSidebar
	}
	return lightSidebar
}

// pillColor is the color of a row of controls inside a card.
func pillColor(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.Hex("#212C46")
	}
	return ui.Hex("#F1F3F6")
}

// cardShadow is the shadow cards cast.
func cardShadow(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.RGBA(0, 0, 0, 0.45)
	}
	return ui.RGBA(15, 23, 42, 0.07)
}

// cardEdge is the hairline around cards, which the light theme draws with
// its border color and the dark one with a lighter line.
func cardEdge(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.RGBA(255, 255, 255, 0.08)
	}
	return ui.RGBA(15, 23, 42, 0.06)
}
