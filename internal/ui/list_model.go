package ui

import (
	"fmt"

	"github.com/bnema/Gart/internal/config"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ListModel struct {
	Table    table.Model
	Dotfiles map[string]string
}

func InitListModel(config config.Config) ListModel {
	fmt.Printf("Dotfiles: %v\n", config.Dotfiles)

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
		Table:    t,
		Dotfiles: config.Dotfiles,
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.Table.Focused() {
				return m, tea.Quit
			}
		}
	}

	m.Table, cmd = m.Table.Update(msg)
	return m, cmd
}

func (m ListModel) View() string {
	return m.Table.View()
}
