package terminalstyle

import (
	"io"
	"os"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

// NewForm creates a Huh form with Dovik's shared terminal presentation.
func NewForm(title string, input io.Reader, output io.Writer, fields ...huh.Field) *huh.Form {
	width, height := formSize(output)
	keymap := NewFormKeyMap()
	form := huh.NewForm(
		huh.NewGroup(fields...).Title(formTitle(title)),
	).
		WithTheme(huhTheme(width)).
		WithKeyMap(keymap).
		WithInput(input).
		WithOutput(output).
		WithViewHook(func(view tea.View) tea.View {
			view.BackgroundColor = lipgloss.Color(CanvasColor)
			view.ForegroundColor = lipgloss.Color(PrimaryColor)
			return view
		})
	if width > 0 && height > 0 {
		form.WithWidth(width).WithHeight(height)
	}
	return form
}

// NewEmbeddedForm creates a themed form updated and rendered by an existing
// Bubble Tea program instead of owning terminal IO or alternate-screen state.
func NewEmbeddedForm(title string, width, height int, fields ...huh.Field) *huh.Form {
	group := huh.NewGroup(fields...)
	if title != "" {
		group = group.Title(formTitle(title))
	}
	form := huh.NewForm(group).
		WithTheme(huhTheme(width)).
		WithKeyMap(NewFormKeyMap()).
		WithShowHelp(false)
	if width > 0 {
		form.WithWidth(width)
	}
	if height > 0 {
		form.WithHeight(height)
	}
	return form
}

// NewFormKeyMap keeps form navigation consistent with the TUI and omits vim
// aliases that would interfere with ordinary text entry.
func NewFormKeyMap() *huh.KeyMap {
	keymap := huh.NewDefaultKeyMap()
	keymap.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"), key.WithHelp("esc", "cancel"))
	keymap.Select.Up = key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up"))
	keymap.Select.Down = key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down"))
	keymap.MultiSelect.Up = key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up"))
	keymap.MultiSelect.Down = key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down"))
	return keymap
}

func formSize(output io.Writer) (int, int) {
	outputFile, ok := output.(*os.File)
	if !ok || !term.IsTerminal(outputFile.Fd()) {
		return 0, 0
	}
	width, height, err := term.GetSize(outputFile.Fd())
	if err != nil {
		return 0, 0
	}
	return max(1, width-1), max(1, height-1)
}

func huhTheme(width int) huh.Theme {
	return huh.ThemeFunc(func(bool) *huh.Styles {
		theme := huh.ThemeBase(true)
		canvas := lipgloss.Color(CanvasColor)
		surface := lipgloss.Color(SurfaceColor)
		primary := lipgloss.Color(PrimaryColor)
		muted := lipgloss.Color(MutedColor)
		accent := lipgloss.Color(AccentColor)
		success := lipgloss.Color(SuccessColor)
		danger := lipgloss.Color(DangerColor)
		border := lipgloss.Color(BorderColor)

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
		theme.Focused.SelectSelector = lipgloss.NewStyle().Foreground(accent).SetString("› ")
		theme.Focused.MultiSelectSelector = theme.Focused.SelectSelector
		theme.Focused.Option = lipgloss.NewStyle().Foreground(primary)
		theme.Focused.SelectedOption = lipgloss.NewStyle().Foreground(primary)
		theme.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(primary)
		theme.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(success).SetString("[x] ")
		theme.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(muted).SetString("[ ] ")
		theme.Focused.NextIndicator = lipgloss.NewStyle().Foreground(accent).SetString("›")
		theme.Focused.PrevIndicator = lipgloss.NewStyle().Foreground(accent).SetString("‹")
		theme.Focused.FocusedButton = lipgloss.NewStyle().Bold(true).Foreground(canvas).Background(accent).Padding(0, 2).MarginRight(1)
		theme.Focused.BlurredButton = lipgloss.NewStyle().Foreground(primary).Background(surface).Padding(0, 2).MarginRight(1)
		theme.Focused.Next = theme.Focused.FocusedButton
		theme.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(accent)
		theme.Focused.TextInput.CursorText = lipgloss.NewStyle().Foreground(canvas).Background(accent)
		theme.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(muted)
		theme.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(accent)
		theme.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(primary).Background(surface).Padding(0, 1)

		theme.Blurred = theme.Focused
		theme.Blurred.Base = theme.Focused.Base.BorderStyle(lipgloss.HiddenBorder()).BorderForeground(border)
		theme.Blurred.Card = theme.Blurred.Base
		theme.Blurred.Title = lipgloss.NewStyle().Bold(true).Foreground(muted)
		theme.Blurred.NoteTitle = theme.Blurred.Title
		theme.Blurred.Description = lipgloss.NewStyle().Foreground(muted)
		theme.Blurred.SelectSelector = lipgloss.NewStyle().Foreground(accent).SetString("• ")
		theme.Blurred.MultiSelectSelector = theme.Blurred.SelectSelector
		theme.Blurred.SelectedOption = lipgloss.NewStyle().Bold(true).Foreground(primary)
		theme.Blurred.NextIndicator = lipgloss.NewStyle()
		theme.Blurred.PrevIndicator = lipgloss.NewStyle()
		theme.Blurred.TextInput.Prompt = lipgloss.NewStyle().Foreground(muted)
		theme.Blurred.TextInput.Text = lipgloss.NewStyle().Foreground(muted).Background(surface).Padding(0, 1)

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

func formTitle(label string) string {
	background := lipgloss.Color(SurfaceColor)
	brand := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(AccentColor)).Background(background).Render(" dovik ")
	title := lipgloss.NewStyle().Foreground(lipgloss.Color(MutedColor)).Background(background).Render(" " + label)
	return brand + title
}
