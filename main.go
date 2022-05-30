package main

import (
	utils "weather-tui/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	t := textinput.NewModel()
	t.Focus()

	s := spinner.NewModel()
	s.Spinner = spinner.Dot

	initialModel := &Model{
		textInput: t,
		spinner:   s,
		typing:    true,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
	err := tea.NewProgram(initialModel, tea.WithAltScreen()).Start()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

type Model struct {
	textInput textinput.Model
	spinner   spinner.Model

	typing  bool
	loading bool
	err     error

	citydata CityData
	client   *http.Client
}

type CityData struct {
	Name string   `json:"name"`
	Main MainData `json:"main"`
	Err  error
}

type MainData struct {
	Temp float64 `json:"temp"`
}

func (m Model) fetchWeather(query string) tea.Cmd {
	return func() tea.Msg {
		// loc, err := m.metaWeather.LocationByQuery(context.Background(), query)

		/////////////////////////////////////////////////////
		resp, err := m.client.Get(utils.GetApiUrl(query))
		var data CityData

		if err != nil {
			fmt.Println(err.Error())
		}

		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&data)

		// fmt.Printf("Data Object: %v\n", data)
		// fmt.Println(data.Name)
		// fmt.Println(data.Main.Temp)
		// fmt.Printf("City %s Temp is %v Degree Celsius\n", data.Name, data.Main.Temp)
		////////////////////////////////////////////////////
		if err != nil {
			return CityData{Err: err}
		}

		return CityData{Name: data.Name, Main: data.Main}
	}
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
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
				m.textInput.Reset()
				m.typing = true
				m.err = nil
				return m, nil
			}

		}

	case CityData:
		m.loading = false
		if err := msg.Err; err != nil {
			m.err = err
			return m, nil
		}

		m.citydata.Name = msg.Name
		m.citydata.Main = msg.Main
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

func (m *Model) View() string {
	if m.typing {
		return fmt.Sprintf("Enter Location:\n%s", m.textInput.View())
	}

	if m.loading {
		return fmt.Sprintf("%s fetching weather for you", m.spinner.View())
	}

	if err := m.err; err != nil {
		return fmt.Sprintf("Could not fetch weather: %v\n", err)
	}

	return fmt.Sprintf("Current Temperature in %s is %v Degree Celsius\n\nPress q to Quit, Esc for searching weather of a different city\n", m.citydata.Name, m.citydata.Main.Temp)
}
