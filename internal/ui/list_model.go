package ui

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var defaultFooter = unchangedStyle.Render("Press 'r' to remove a dotfile or 'q' to quit")

type defaultFooterMsg struct{}

type ListModel struct {
	App           *app.App
	Table         table.Model
	KeyMap        KeyMap
	Dotfile       app.Dotfile
	Dotfiles      map[string]string
	Footer        string
	ConfirmRemove bool
}

type KeyMap struct {
	Remove key.Binding
	Quit   key.Binding
	Esc    key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Remove: key.NewBinding(
			key.WithKeys("r", "R"),
			key.WithHelp("r", "remove a dotfile"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "Q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "quit"),
		),
	}
}

func RunListView(app *app.App) {
	dotfiles := app.GetDotfiles()
	if len(dotfiles) == 0 {
		fmt.Println("No dotfiles found. Please add some dotfiles first.")
		return
	}

	model := InitListModel(*app.Config, app)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func InitListModel(config config.Config, app *app.App) ListModel {
	var rows []table.Row
	for name, path := range config.Dotfiles {
		rows = append(rows, table.Row{name, path})
	}

	// Sort the rows alphabetically by the Dotfiles column (index 0)
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})

	// Count the number of rows
	rowCount := len(rows)

	columns := []table.Column{
		{Title: fmt.Sprintf("Dotfiles (%d)", rowCount), Width: 15},
		{Title: unchangedStyle.Render("Origin Path"), Width: 50},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return ListModel{
		App:           app,
		Table:         t,
		KeyMap:        DefaultKeyMap(),
		Dotfiles:      config.Dotfiles,
		Footer:        defaultFooter,
		ConfirmRemove: false,
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}
func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.KeyMap.Esc):
			if m.ConfirmRemove {
				m.ConfirmRemove = false
				m.Footer = unchangedStyle.Render("Removal cancelled. Press 'r' to remove a dotfile or 'q' to quit")
			} else {
				return m, tea.Quit
			}
		case key.Matches(msg, m.KeyMap.Remove):
			if !m.ConfirmRemove {
				m.ConfirmRemove = true
				m.Footer = alertStyle.Render("Are you sure you want to remove the selected dotfile? (y/n)")
			}
		case msg.String() == "y" || msg.String() == "Y":
			if m.ConfirmRemove {
				return m.removeSelectedEntry()
			}
		case msg.String() == "n" || msg.String() == "N":
			if m.ConfirmRemove {
				m.ConfirmRemove = false
				m.Footer = unchangedStyle.Render("Removal cancelled. Press 'r' to remove a dotfile or 'q' to quit")
			}
		}
	case defaultFooterMsg:
		m.Footer = defaultFooter
	}

	m.Table, cmd = m.Table.Update(msg)
	return m, cmd
}

func (m ListModel) removeSelectedEntry() (tea.Model, tea.Cmd) {
	selectedRow := m.Table.SelectedRow()
	if len(selectedRow) > 0 {
		name := selectedRow[0]
		path := selectedRow[1]

		// Remove the selected entry from the config
		err := m.App.RemoveDotFile(path, name)
		if err != nil {
			m.Footer = errorStyle.Render(fmt.Sprintf("Error removing dotfile: %s", err))
			return m, nil
		}

		// Remove the row from the table
		rows := m.Table.Rows()
		for i, row := range rows {
			if row[0] == name && row[1] == path {
				m.Table.SetRows(append(rows[:i], rows[i+1:]...))
				break
			}
		}

		// Update the column title with the new count
		m.updateNameColumnTitle()

		m.ConfirmRemove = false
		m.Footer = successStyle.Render(fmt.Sprintf("Dotfile '%s' removed successfully", name))

		// Return the model with a command to clear the footer after 3 seconds
		return m, clearFooterAfter(3 * time.Second)
	}

	return m, nil
}

func clearFooterAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return defaultFooterMsg{}
	})
}

func (m ListModel) View() string {
	tableView := m.Table.View()
	footer := m.Footer

	// Get the height of the terminal
	height := m.Table.Height()

	// Calculate the position of the footer
	footerPosition := height - 1

	// Create a view with the table and the footer at the bottom
	view := lipgloss.JoinVertical(lipgloss.Left, tableView, lipgloss.Place(footerPosition, 0, lipgloss.Left, lipgloss.Bottom, footer))

	return view
}

func (m *ListModel) updateNameColumnTitle() {
	m.Table.SetColumns([]table.Column{
		{Title: fmt.Sprintf("Dotfiles (%d)", len(m.Table.Rows())), Width: 15},
		{Title: unchangedStyle.Render("Origin Path"), Width: 50},
	})
}
