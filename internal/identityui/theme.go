package identityui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/MrMaxie/dovik/internal/terminalstyle"
)

func dovikTheme(width int) huh.Theme {
	return huh.ThemeFunc(func(bool) *huh.Styles {
		theme := huh.ThemeBase(true)
		canvas := lipgloss.Color(terminalstyle.CanvasColor)
		surface := lipgloss.Color(terminalstyle.SurfaceColor)
		primary := lipgloss.Color(terminalstyle.PrimaryColor)
		muted := lipgloss.Color(terminalstyle.MutedColor)
		accent := lipgloss.Color(terminalstyle.AccentColor)
		success := lipgloss.Color(terminalstyle.SuccessColor)
		danger := lipgloss.Color(terminalstyle.DangerColor)
		border := lipgloss.Color(terminalstyle.BorderColor)

		theme.Form.Base = theme.Form.Base.Foreground(primary).Background(canvas)
		theme.Group.Base = theme.Group.Base.Foreground(primary).Background(canvas)
		theme.Group.Title = lipgloss.NewStyle().Background(surface)
		if width > 0 {
			theme.Group.Title = theme.Group.Title.Width(width)
		}
		theme.Group.Description = lipgloss.NewStyle().Foreground(muted)
		theme.FieldSeparator = lipgloss.NewStyle().SetString("\n")

		theme.Focused.Base = lipgloss.NewStyle().PaddingLeft(1).BorderStyle(lipgloss.NormalBorder()).BorderLeft(true).BorderForeground(accent)
		theme.Focused.Card = theme.Focused.Base
		theme.Focused.Title = lipgloss.NewStyle().Bold(true).Foreground(primary)
		theme.Focused.NoteTitle = theme.Focused.Title
		theme.Focused.Description = lipgloss.NewStyle().Foreground(muted)
		theme.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(danger).SetString(" !")
		theme.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(danger).SetString(" !")
		theme.Focused.SelectSelector = lipgloss.NewStyle().Foreground(accent).SetString("> ")
		theme.Focused.MultiSelectSelector = theme.Focused.SelectSelector
		theme.Focused.Option = lipgloss.NewStyle().Foreground(primary)
		theme.Focused.SelectedOption = lipgloss.NewStyle().Foreground(primary)
		theme.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(primary)
		theme.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(success).SetString("[x] ")
		theme.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(muted).SetString("[ ] ")
		theme.Focused.NextIndicator = lipgloss.NewStyle().Foreground(accent).SetString(">")
		theme.Focused.PrevIndicator = lipgloss.NewStyle().Foreground(accent).SetString("<")
		theme.Focused.FocusedButton = lipgloss.NewStyle().Bold(true).Foreground(canvas).Background(accent).Padding(0, 2).MarginRight(1)
		theme.Focused.BlurredButton = lipgloss.NewStyle().Foreground(primary).Background(surface).Padding(0, 2).MarginRight(1)
		theme.Focused.Next = theme.Focused.FocusedButton
		theme.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(accent)
		theme.Focused.TextInput.CursorText = lipgloss.NewStyle().Foreground(canvas).Background(accent)
		theme.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(muted)
		theme.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(accent)
		theme.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(primary)

		theme.Blurred = theme.Focused
		theme.Blurred.Base = theme.Focused.Base.BorderStyle(lipgloss.HiddenBorder()).BorderForeground(border)
		theme.Blurred.Card = theme.Blurred.Base
		theme.Blurred.Title = lipgloss.NewStyle().Bold(true).Foreground(muted)
		theme.Blurred.NoteTitle = theme.Blurred.Title
		theme.Blurred.Description = lipgloss.NewStyle().Foreground(muted)
		theme.Blurred.SelectSelector = lipgloss.NewStyle().SetString("  ")
		theme.Blurred.MultiSelectSelector = theme.Blurred.SelectSelector
		theme.Blurred.NextIndicator = lipgloss.NewStyle()
		theme.Blurred.PrevIndicator = lipgloss.NewStyle()
		theme.Blurred.TextInput.Prompt = lipgloss.NewStyle().Foreground(muted)

		theme.Help.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(accent)
		theme.Help.ShortDesc = lipgloss.NewStyle().Foreground(muted)
		theme.Help.ShortSeparator = lipgloss.NewStyle().Foreground(border)
		theme.Help.FullKey = theme.Help.ShortKey
		theme.Help.FullDesc = theme.Help.ShortDesc
		theme.Help.FullSeparator = theme.Help.ShortSeparator
		theme.Help.Ellipsis = theme.Help.ShortSeparator
		return theme
	})
}

func questionnaireTitle() string {
	background := lipgloss.Color(terminalstyle.SurfaceColor)
	brand := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(terminalstyle.AccentColor)).Background(background).Render(" dovik ")
	title := lipgloss.NewStyle().Foreground(lipgloss.Color(terminalstyle.MutedColor)).Background(background).Render(" Configure project identity")
	return brand + title
}
