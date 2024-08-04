package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type AddModel struct {
	FocusIndex int
	Inputs     []textinput.Model
}

func (m AddModel) Init() tea.Cmd {
	return textinput.Blink
}

func InitAddModel() AddModel {
	m := AddModel{
		Inputs: make([]textinput.Model, 2),
	}

	var t textinput.Model
	for i := range m.Inputs {
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

		m.Inputs[i] = t
	}

	return m
}

func (m AddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyTab, tea.KeyShiftTab:
			s := msg.String()
			if s == "tab" {
				m.FocusIndex++
				if m.FocusIndex > len(m.Inputs) {
					m.FocusIndex = 0
				}
			} else if s == "shift+tab" {
				m.FocusIndex--
				if m.FocusIndex < 0 {
					m.FocusIndex = len(m.Inputs)
				}
			}
			cmds := make([]tea.Cmd, len(m.Inputs))
			for i := 0; i < len(m.Inputs); i++ {
				if i == m.FocusIndex {
					cmds[i] = m.Inputs[i].Focus()
					m.Inputs[i].PromptStyle = focusedStyle
					m.Inputs[i].TextStyle = focusedStyle
				} else {
					m.Inputs[i].Blur()
					m.Inputs[i].PromptStyle = noStyle
					m.Inputs[i].TextStyle = noStyle
				}
			}
			return m, tea.Batch(cmds...)

		case tea.KeyEnter:
			if m.FocusIndex == len(m.Inputs) {
				if m.Inputs[0].Value() != "" && m.Inputs[1].Value() != "" {
					return m, tea.Quit
				}
			} else {
				m.FocusIndex++
				if m.FocusIndex > len(m.Inputs) {
					m.FocusIndex = 0
				}
				cmds := make([]tea.Cmd, len(m.Inputs))
				for i := 0; i < len(m.Inputs); i++ {
					if i == m.FocusIndex {
						cmds[i] = m.Inputs[i].Focus()
						m.Inputs[i].PromptStyle = focusedStyle
						m.Inputs[i].TextStyle = focusedStyle
					} else {
						m.Inputs[i].Blur()
						m.Inputs[i].PromptStyle = noStyle
						m.Inputs[i].TextStyle = noStyle
					}
				}
				return m, tea.Batch(cmds...)
			}
		}
	}

	// Update the focused input field or the submit button
	if m.FocusIndex < len(m.Inputs) {
		m.Inputs[m.FocusIndex], cmd = m.Inputs[m.FocusIndex].Update(msg)
	}

	return m, cmd
}

func (m *AddModel) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.Inputs {
		m.Inputs[i], _ = m.Inputs[i].Update(msg)
		cmds = append(cmds, textinput.Blink)
	}
	return tea.Batch(cmds...)
}

func (m AddModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("Add a dotfile location\n\n")

	for i := range m.Inputs {
		b.WriteString(m.Inputs[i].View())
		if i < len(m.Inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.FocusIndex == len(m.Inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	return b.String()
}
