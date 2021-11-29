package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicolasparada/go-tea-weather/metaweather"
)

func main() {
	t := textinput.NewModel()
	t.Focus()

	s := spinner.NewModel()
	s.Spinner = spinner.Dot

	initialModel := Model{
		textInput:   t,
		spinner:     s,
		typing:      true,
		metaWeather: &metaweather.Client{HTTPClient: http.DefaultClient},
	}
	err := tea.NewProgram(initialModel, tea.WithAltScreen()).Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type Model struct {
	textInput   textinput.Model
	spinner     spinner.Model
	metaWeather *metaweather.Client

	typing   bool
	loading  bool
	err      error
	location metaweather.Location
}

type GotWeather struct {
	Err      error
	Location metaweather.Location
}

func (m Model) fetchWeather(query string) tea.Cmd {
	return func() tea.Msg {
		loc, err := m.metaWeather.LocationByQuery(context.Background(), query)
		if err != nil {
			return GotWeather{Err: err}
		}

		return GotWeather{Location: loc}
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.typing {
				query := strings.TrimSpace(m.textInput.Value())
				if query != "" {
					m.typing = false
					m.loading = true
					return m, tea.Batch(
						spinner.Tick,
						m.fetchWeather(query),
					)
				}
			}

		case "esc":
			if !m.typing && !m.loading {
				m.typing = true
				m.err = nil
				return m, nil
			}
		}

	case GotWeather:
		m.loading = false

		if err := msg.Err; err != nil {
			m.err = err
			return m, nil
		}

		m.location = msg.Location
		return m, nil
	}

	if m.typing {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.typing {
		return fmt.Sprintf("Enter location:\n%s", m.textInput.View())
	}

	if m.loading {
		return fmt.Sprintf("%s fetching weather... please wait.", m.spinner.View())
	}

	if err := m.err; err != nil {
		return fmt.Sprintf("Could not fetch weather: %v", err)
	}

	return fmt.Sprintf("Current weather in %s is %.0f °C", m.location.Title, m.location.ConsolidatedWeather[0].TheTemp)
}
