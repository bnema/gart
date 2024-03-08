package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnema/Gart/config"
	"github.com/bnema/Gart/system"
	"github.com/bnema/Gart/utils"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	focusedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	noStyle       = lipgloss.NewStyle()
	helpStyle     = blurredStyle.Copy()
	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type App struct {
	ListModel      listModel
	AddModel       addModel
	ConfigFilePath string
	StorePath      string
}

type addModel struct {
	focusIndex int
	inputs     []textinput.Model
}

type listModel struct {
	table    table.Model
	dotfiles map[string]string
}

func initialAddModel() addModel {
	m := addModel{
		inputs: make([]textinput.Model, 2),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.CharLimit = 64

		switch i {
		case 0:
			t.Placeholder = "Directory Path"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "Name"
		}

		m.inputs[i] = t
	}

	return m
}

func (m addModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyTab, tea.KeyShiftTab:
			s := msg.String()
			if s == "tab" {
				m.focusIndex++
				if m.focusIndex > len(m.inputs) {
					m.focusIndex = 0
				}
			} else if s == "shift+tab" {
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs)
				}
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
				} else {
					m.inputs[i].Blur()
					m.inputs[i].PromptStyle = noStyle
					m.inputs[i].TextStyle = noStyle
				}
			}
			return m, tea.Batch(cmds...)

		case tea.KeyEnter:
			if m.focusIndex == len(m.inputs) {
				if m.inputs[0].Value() != "" && m.inputs[1].Value() != "" {
					return m, tea.Quit
				}
			} else {
				m.focusIndex++
				if m.focusIndex > len(m.inputs) {
					m.focusIndex = 0
				}
				cmds := make([]tea.Cmd, len(m.inputs))
				for i := 0; i < len(m.inputs); i++ {
					if i == m.focusIndex {
						cmds[i] = m.inputs[i].Focus()
						m.inputs[i].PromptStyle = focusedStyle
						m.inputs[i].TextStyle = focusedStyle
					} else {
						m.inputs[i].Blur()
						m.inputs[i].PromptStyle = noStyle
						m.inputs[i].TextStyle = noStyle
					}
				}
				return m, tea.Batch(cmds...)
			}
		}
	}

	// Update the focused input field or the submit button
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	}

	return m, cmd
}

func (m *addModel) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		m.inputs[i], _ = m.inputs[i].Update(msg)
		cmds = append(cmds, textinput.Blink)
	}
	return tea.Batch(cmds...)
}

func (m addModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("Add a dotfile location\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	return b.String()
}

func initialListModel(app *App) listModel {
	dotfiles := make(map[string]string)
	config.LoadConfig(app.ConfigFilePath, dotfiles)

	var rows []table.Row
	for name, path := range dotfiles {
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

	return listModel{
		table:    t,
		dotfiles: dotfiles,
	}
}

func (m listModel) Init() tea.Cmd {
	return nil
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.table.Focused() {
				return m, tea.Quit
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	return m.table.View()
}

func (app *App) runAddForm() {
	model := initialAddModel()
	if finalModel, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	} else {
		finalAddModel, ok := finalModel.(addModel)
		if ok {
			if finalAddModel.inputs[0].Value() != "" && finalAddModel.inputs[1].Value() != "" {
				app.addDotfile(finalAddModel.inputs[0].Value(), finalAddModel.inputs[1].Value())
			}
		}
	}
}

func (app *App) runUpdateView(name, path string) {
	storePath := filepath.Join(app.StorePath, name)
	changed, err := utils.DiffFiles(path, storePath)
	if err != nil {
		fmt.Printf("Error comparing dotfiles: %v\n", err)
		return
	}

	if changed {
		fmt.Printf("Changes detected in '%s'. Saving the updated dotfiles.\n", name)
		// Logic to save the modified files
	} else {
		fmt.Printf("No changes detected in '%s' since the last update.\n", name)
	}
}

func (app *App) runListView() {
	if _, err := tea.NewProgram(app.ListModel).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func (app *App) addDotfile(path, name string) {
	// Get the user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}

	// Remove the tilde from the path
	cleanedPath := strings.TrimPrefix(path, "~")

	// Expand the path
	expandedPath := filepath.Join(home, cleanedPath)

	storePath := filepath.Join(app.StorePath, name)
	err = system.CopyDirectory(expandedPath, storePath)
	if err != nil {
		fmt.Printf("Error copying directory: %v\n", err)
		return
	}

	app.ListModel.dotfiles[name] = expandedPath
	config.SaveConfig(app.ConfigFilePath, app.ListModel.dotfiles)
	fmt.Printf("Dotfile added: %s\n", name)
}

func main() {
	configPath := utils.GetOSConfigPath()

	app := &App{
		ConfigFilePath: filepath.Join(configPath, "config.toml"),
		StorePath:      filepath.Join(configPath, ".store"),
	}
	app.ListModel = initialListModel(app)

	var rootCmd = &cobra.Command{
		Use:   "gart",
		Short: "Gart is a dotfile manager",
		Long:  `Gart is a command-line tool for managing dotfiles.`,
	}

	// Show at root level the list of dotfiles
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all dotfiles",
		Run: func(cmd *cobra.Command, args []string) {
			app.runListView()
		},
	}

	var addCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new dotfile",
		Run: func(cmd *cobra.Command, args []string) {
			app.runAddForm()
		},
	}

	var updateCmd = &cobra.Command{
		Use:   "update [name]",
		Short: "Update a dotfile",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// Update all dotfiles
				for name, path := range app.ListModel.dotfiles {
					app.runUpdateView(name, path)
				}
			} else {
				// Update a specific dotfile
				name := args[0]
				path, ok := app.ListModel.dotfiles[name]
				if !ok {
					fmt.Printf("Dotfile '%s' not found.\n", name)
					return
				}
				app.runUpdateView(name, path)
			}
		},
	}

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(listCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
