package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	noStyle      = lipgloss.NewStyle()
	helpStyle    = blurredStyle.Copy()

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type model struct {
	table      table.Model
	focusIndex int
	inputs     []textinput.Model
	dotfiles   map[string]string
	tooltip    string
}

func initialModel() model {
	dotfiles := make(map[string]string)
	loadConfig(dotfiles)

	m := model{
		inputs:   make([]textinput.Model, 2),
		dotfiles: dotfiles,
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

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Quit,
			)
		}
	}

	// Update the inputs
	inputCmd := m.updateInputs(msg)

	// Update the table
	var tableCmd tea.Cmd
	m.table, tableCmd = m.table.Update(msg)

	// Combine the input and table commands
	cmd = tea.Batch(inputCmd, tableCmd)

	return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		m.inputs[i], _ = m.inputs[i].Update(msg)
		cmds = append(cmds, textinput.Blink)
	}
	return tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString("Add a dotfile location\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	table := m.table.View()
	b.WriteString("\n\n")
	b.WriteString(table)

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	return b.String()
}

func (m *model) updateTable() {
	var rows []table.Row
	for name, path := range m.dotfiles {
		rows = append(rows, table.Row{name, path})
	}
	m.table.SetRows(rows)
}

func (m *model) addDotfile(path, name string) {
	m.dotfiles[name] = path
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

	storePath := filepath.Join(".store", name)
	err = copyDirectory(expandedPath, storePath)
	if err != nil {
		fmt.Printf("Error copying directory: %v\n", err)
		return
	}

	m.dotfiles[name] = expandedPath
	fmt.Printf("Dotfile added: %s\n", name)
	m.updateTable()
	saveConfig(m.dotfiles)
}

func loadConfig(dotfiles map[string]string) {
	configPath := "config.toml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Printf("Failed to read config file: %v\n", err)
		return
	}

	var config map[string]interface{}
	err = toml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("Failed to parse config file: %v\n", err)
		return
	}

	for key, value := range config {
		if path, ok := value.(string); ok {
			dotfiles[key] = path
		}
	}
}

func saveConfig(dotfiles map[string]string) {
	data, err := toml.Marshal(dotfiles)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		return
	}

	err = os.WriteFile("config.toml", data, 0644)
	if err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}
}

func copyDirectory(src, dst string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return err
	}

	// Read the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Skip symlinks
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			// Copy regular files
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the file contents
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Preserve the file mode
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

func diffFiles(path1, path2 string) (bool, error) {
	dmp := diffmatchpatch.New()

	var diff func(string, string) (bool, error)
	diff = func(p1, p2 string) (bool, error) {
		info1, err := os.Stat(p1)
		if err != nil {
			return false, err
		}

		info2, err := os.Stat(p2)
		if os.IsNotExist(err) {
			// Le fichier existe dans path1 mais pas dans path2
			return true, nil
		} else if err != nil {
			return false, err
		}

		if info1.IsDir() && info2.IsDir() {
			files1, err := os.ReadDir(p1)
			if err != nil {
				return false, err
			}

			changed := false
			for _, file1 := range files1 {
				filePath1 := filepath.Join(p1, file1.Name())
				filePath2 := filepath.Join(p2, file1.Name())

				fileChanged, err := diff(filePath1, filePath2)
				if err != nil {
					return false, err
				}
				if fileChanged {
					changed = true
				}
			}
			return changed, nil
		} else if !info1.IsDir() && !info2.IsDir() {
			content1, err := os.ReadFile(p1)
			if err != nil {
				return false, err
			}

			content2, err := os.ReadFile(p2)
			if err != nil {
				return false, err
			}

			diffs := dmp.DiffMain(string(content1), string(content2), false)
			return len(diffs) > 1, nil
		}

		return false, nil
	}

	changed, err := diff(path1, path2)
	if err != nil {
		return false, err
	}
	return changed, nil
}

func runAddForm() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func runUpdateView(name string) {
	dotfiles := make(map[string]string)
	loadConfig(dotfiles)

	path, ok := dotfiles[name]
	if !ok {
		fmt.Printf("Dotfile '%s' not found.\n", name)
		return
	}

	storePath := filepath.Join(".store", name)
	changed, err := diffFiles(path, storePath)
	if err != nil {
		fmt.Printf("Error comparing dotfiles: %v\n", err)
		return
	}

	if changed {
		fmt.Println("Changes detected. Saving the updated dotfiles.")
		// Logique pour sauvegarder les fichiers modifiés
	} else {
		fmt.Println("No changes detected since the last update.")
	}
}

func runListView(dotfiles map[string]string) {
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

	m := model{
		table:      t,
		focusIndex: 0,
		dotfiles:   dotfiles,
		tooltip:    "Press 'q' or 'ctrl+c' to quit",
	}
	// Run the program
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func main() {
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
			dotfiles := make(map[string]string)
			loadConfig(dotfiles)
			runListView(dotfiles)
		},
	}

	var addCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new dotfile",
		Run: func(cmd *cobra.Command, args []string) {
			runAddForm()
		},
	}

	var updateCmd = &cobra.Command{
		Use:   "update [name]",
		Short: "Update a dotfile",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			runUpdateView(name)
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
