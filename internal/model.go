package internal

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type TickMsg time.Time

type Model struct {
	// Keys holds key bindings.
	Keys KeyMap
	// Obstacles is a data structure containing obstacles.
	Obstacles Obstacles
	// Cursor is the location of the cursor.
	Cursor Location
	// Score is the number of obstacles avoided.
	Score int
	// Help contains the Bubble Tea help model.
	Help help.Model
	// Viewport is the size of the game area.
	Viewport Location
	// Over is true when the player has lost.
	Over bool
	// Pressed is used to lock the cursor from moving until the next tick.
	Pressed bool
	// Layouted tracks whether the initial layout has been performed.
	Layouted bool
}

func NewModel() Model {
	return Model{
		Keys: KeyMap{
			Up:   key.NewBinding(key.WithKeys("k", "up", " ", "w"), key.WithHelp("↑/k/w/espace", "sauter")),
			Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q/ctrl+c", "quitter")),
		},
		Obstacles: NewObstacles(),
		Cursor:    Location{},
		Score:     0,
		Help:      help.New(),
		Over:      false,
		Viewport:  Location{},
		Pressed:   false,
		Layouted:  false,
	}
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(time.Second/5, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return m.tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.Keys.Quit):
			fmt.Println()
			return m, tea.Quit

		case key.Matches(msg, m.Keys.Up):
			if !m.Pressed && m.Cursor.Y > 0 {
				m.Cursor.Y--
			}
			// Disable the key until the next tick.
			// Since the view does not update in real time, this prevents
			// hidden states in the game that are invisible to the user.
			m.Pressed = true
		}
	case tea.WindowSizeMsg:
		// Terminal resized.
		if !m.Layouted {
			m.Layouted = true
			m.Help.Width = msg.Width
			m.Viewport.X = 60
			m.Viewport.Y = 20
			m.Cursor = Location{2, 0}
		}
	case TickMsg:
		return m.Frame()
	}
	return m, nil
}

func (m Model) Frame() (tea.Model, tea.Cmd) {
	if !m.Pressed {
		m.Cursor.Y++
	}
	m.Pressed = false

	for _, obst := range m.Obstacles {
		// Shift the obstacle left.
		obst.Location.X--
		// Check if the player has lost.
		if (obst.Collides(m.Cursor)) || m.Cursor.Y >= m.Viewport.Y {
			m.Over = true
			return m, tea.Quit
		}
		if obst.Location.X == m.Cursor.X {
			// Check if the player has earned a point.
			m.Score++
		} else if obst.Location.X < 0 {
			// Clean up the obstacle if it has left the viewport.
			m.Obstacles.Remove()
		}
	}

	var rightmost *Location
	if len(m.Obstacles) > 0 {
		rightmost = m.Obstacles[len(m.Obstacles)-1].Location
	} else {
		rightmost = &Location{0, m.Viewport.Y / 2}
	}
	gap := m.Viewport.X - rightmost.X 
	if gap > 5 || (rand.Intn(100) > 90 && gap > 2) {
		x := m.Viewport.X
		// Select a y that makes the obstacle possible to avoid.
		var y int
		for {
			y = rand.Intn(m.Viewport.Y)
			if abs(y-rightmost.Y) < gap {
				break
			}
		}
		// Clamp y to the viewport.
		if y < 0 {
			y = 0
		} else if y >= m.Viewport.Y {
			y = m.Viewport.Y - 1
		}
		m.Obstacles.Add(NewObstacle(DefaultAperture, &Location{x, y}))
	}
	return m, m.tick()
}

func (m Model) View() string {
	var sb strings.Builder
	sb.WriteString(TitleStyle.Render("FlappIssou"))
	sb.WriteByte('\n')

	viewport := make([]string, 0, m.Viewport.Y)
	for y := 0; y < m.Viewport.Y; y++ {
		var line strings.Builder
		// Store the index of the leftmost obstacle encountered.
		// This is used to slice the obstacle list to avoid checking obstacles
		// we've already seen.
		leftmost := 0
		for x := 0; x < m.Viewport.X; x++ {
			// Check if any obstacles collide with this cell.
			cellValue := ' '
			for _, o := range m.Obstacles[leftmost:] {
				if o.Collides(Location{x, y}) {
					cellValue = '#'
					leftmost++
					break
				}
			}
			if m.Cursor.X == x && m.Cursor.Y == y {
				cellValue = '*'
			}
			line.WriteRune(cellValue)
		}
		viewport = append(viewport, line.String())
	}

	sb.WriteString(ViewportStyle.Render(strings.Join(viewport, "\n")))
	sb.WriteString(fmt.Sprintf("\nSCORE : %d - ", m.Score))
	sb.WriteString(m.Help.View(m.Keys))

	if m.Over {
		sb.WriteString(GameOverStyle.Render("\n\n> Issou ! <"))
	}

	// Send the UI for rendering
	return ViewStyle.Render(sb.String())
}
