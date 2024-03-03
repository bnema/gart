package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type model struct {
	watchedFolder string
	cacheFolder   string
	fileName      string
	lastModified  map[string]time.Time
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "gart",
		Short: "Gart monitors your configuration files and copies changes to a cache folder",
	}

	var watchCmd = &cobra.Command{
		Use:   "watch [folder] -n [name]",
		Short: "Monitors a configuration folder",
		Args:  cobra.ExactArgs(1),
		Run:   watchFolder,
	}

	watchCmd.Flags().StringP("name", "n", "", "Name of the configuration to monitor")

	rootCmd.AddCommand(watchCmd)
	rootCmd.Execute()
}

func watchFolder(cmd *cobra.Command, args []string) {
	folder := args[0]
	name, _ := cmd.Flags().GetString("name")

	initialModel := model{
		watchedFolder: folder,
		cacheFolder:   "./cache", 
		fileName:      name,
		lastModified:  make(map[string]time.Time),
	}

	p := tea.NewProgram(initialModel)
	_, err := p.Run()
	if err != nil {
		log.Panic(err)
	}
}

func (m model) Init() tea.Cmd {
	return m.watchFiles
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	case tea.WindowSizeMsg:
		return m, nil
	}

	return m, m.watchFiles
}

func (m model) View() string {
	return fmt.Sprintf("You are monitoring the folder '%s' for files '%s'", m.watchedFolder, m.fileName)
}

func (m model) watchFiles() tea.Msg {
	files, err := os.ReadDir(m.watchedFolder)
	if err != nil {
		log.Printf("Error reading the folder: %v", err)
		return nil
	}

	for _, file := range files {
		filePath := filepath.Join(m.watchedFolder, file.Name())
		if fileChanged(filePath, m.lastModified[file.Name()]) {
			destPath := filepath.Join(m.cacheFolder, file.Name())
			err := copyFile(filePath, destPath)
			if err != nil {
				log.Printf("Error copying the file: %v", err)
			} else {
				m.lastModified[file.Name()] = time.Now()
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

func fileChanged(path string, lastModTime time.Time) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.ModTime().After(lastModTime)
}
