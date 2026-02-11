package tui

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	_ tea.Model = (*App)(nil)
	_           = progress.Model{}
	_           = lipgloss.Style{}
)

type App struct{}

func (a *App) Init() tea.Cmd                       { return nil }
func (a *App) Update(tea.Msg) (tea.Model, tea.Cmd) { return a, nil }
func (a *App) View() string                        { return "" }
