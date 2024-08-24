package ui

import (
	"fmt"
	"time"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const defaultFooter = "Press 'r' to remove a dotfile or 'q' to quit"

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

func InitListModel(config config.Config, app *app.App) ListModel {
	var rows []table.Row
	for name, path := range config.Dotfiles {
		rows = append(rows, table.Row{name, path})
	}

	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Path", Width: 50},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
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
				m.Footer = "Removal cancelled. Press 'r' to remove a dotfile or 'q' to quit"
			} else {
				return m, tea.Quit
			}
		case key.Matches(msg, m.KeyMap.Remove):
			if !m.ConfirmRemove {
				m.ConfirmRemove = true
				m.Footer = "Are you sure you want to remove this dotfile? (y/n)"
			}
		case msg.String() == "y" || msg.String() == "Y":
			if m.ConfirmRemove {
				return m.removeSelectedEntry()
			}
		case msg.String() == "n" || msg.String() == "N":
			if m.ConfirmRemove {
				m.ConfirmRemove = false
				m.Footer = "Removal cancelled. Press 'r' to remove a dotfile or 'q' to quit"
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
