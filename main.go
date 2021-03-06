package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
	utils "weather-tui/utils"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

func main() {
	w, h, err := term.GetSize(0)
	if err != nil {
		return
	}

	t := textinput.NewModel()
	t.Focus()

	s := spinner.NewModel()
	s.Spinner = spinner.Dot

	initialModel := &Model{
		textInput: t,
		spinner:   s,
		typing:    true,
		client:    &http.Client{Timeout: 10 * time.Second},
		width:     w,
		height:    h,
	}
	err = tea.NewProgram(initialModel, tea.WithAltScreen()).Start()

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

	width  int
	height int
}

type CityData struct {
	Name   string   `json:"name"`
	Main   MainData `json:"main"`
	Err    error
	Status string `json:"message,omitempty"`
}

type MainData struct {
	Temp float64 `json:"temp"`
}

func (m Model) fetchWeather(query string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.Get(utils.GetApiUrl(query))
		var data CityData

		if err != nil {
			fmt.Println(err.Error())
		}

		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&data)

		if err != nil {
			return CityData{Err: err}
		}

		return data
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
				m.citydata.Name = ""
				m.citydata.Status = ""
				m.typing = true
				m.err = nil
				return m, nil
			}

		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		return m, nil

	case CityData:
		m.loading = false
		if err := msg.Err; err != nil {
			m.err = err
			return m, nil
		}

		if msg.Status != "" {
			m.err = errors.New(msg.Status)
			return m, nil
		}

		m.citydata.Name = msg.Name
		m.citydata.Main = msg.Main
		m.citydata.Status = msg.Status
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

var subtle = lipgloss.Color("#808080")

var dialogBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#808080")).
	Padding(1, 0).
	BorderTop(true).
	BorderLeft(true).
	BorderRight(true).
	BorderBottom(true)

var locationInputStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#ffffff")).
	Border(lipgloss.RoundedBorder()).
	BorderTop(true).
	BorderLeft(true).
	BorderRight(true).
	BorderBottom(true).
	BorderForeground(lipgloss.Color("#808080")).
	MarginRight(2).
	Padding(0, 3).
	MarginTop(1).Bold(true)

func (m *Model) View() string {
	var ui string
	if m.typing {
		question := lipgloss.NewStyle().
			Width(int(math.Min(float64(m.width), 50))).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#808080")).
			Render("Enter the name of location")
		locationInput := locationInputStyle.Render(m.textInput.Value())
		ui = lipgloss.JoinVertical(lipgloss.Center, question, locationInput)
	}

	if m.loading {
		fetching := lipgloss.NewStyle().
			Width(int(math.Min(float64(m.width), 50))).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#808080")).
			Render("Fetching weather for you")
		loading := locationInputStyle.Render(m.spinner.View())
		ui = lipgloss.JoinVertical(lipgloss.Center, fetching, loading)
	}

	if err := m.err; err != nil {
		fetching := lipgloss.NewStyle().
			Width(int(math.Min(float64(m.width), 50))).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#ff0000")).
			Render(fmt.Sprintf("Could not fetch weather: %v", err))
		ui = lipgloss.JoinVertical(lipgloss.Center, fetching)
	}

	if (m.citydata.Name != "") && (m.citydata.Status == "") {
		cityName := lipgloss.NewStyle().
			Underline(true).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Render(m.citydata.Name)

		resultText := lipgloss.NewStyle().
			Width(int(math.Min(float64(m.width), 50))).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#ffffff")).
			Render(fmt.Sprintf("Current Temperature of %s", cityName))
		cityTemp := locationInputStyle.Render(fmt.Sprintf("%v ??C", m.citydata.Main.Temp))
		ui = lipgloss.JoinVertical(lipgloss.Center, resultText, cityTemp)
	}

	instruction := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("#808080")).
		MarginRight(1).
		Render("\n| Press q or ctrl+c to Quit |\n\n| Press Esc to search weather of a different city |\n")

	completeUi := lipgloss.JoinVertical(lipgloss.Center, dialogBoxStyle.Render(ui), instruction)
	dialog := lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		completeUi,
		lipgloss.WithWhitespaceChars(`?? ??? ??? ??? ??? ??? ??? ??? ??? ???`),
		lipgloss.WithWhitespaceForeground(subtle),
	)
	return dialog
}
