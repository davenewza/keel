package program

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/teamkeel/keel/codegen"
	"github.com/teamkeel/keel/colors"
)

const (
	StatusInitializing = iota
	StatusInitialized
)

type InitModel struct {
	// The directory of the Keel project
	ProjectDir string

	Err    error
	Status int

	GeneratedFiles codegen.GeneratedFiles

	// Maintain the current dimensions of the user's terminal
	width  int
	height int
}

func (m *InitModel) Init() tea.Cmd {
	m.Status = StatusInitializing
	return Init(m.ProjectDir)
}

func (m *InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Status = StatusQuitting
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		// This msg is sent once on program start
		// and then subsequently every time the user
		// resizes their terminal window.
		m.width = msg.Width
		m.height = msg.Height

		return m, nil
	case InitialisedMsg:
		m.Status = StatusInitialized
		m.GeneratedFiles = msg.GeneratedFiles

		if msg.Err != nil {
			m.Err = msg.Err
		}

		return m, tea.Quit
	}

	return m, nil
}

func (m *InitModel) View() string {
	b := strings.Builder{}

	// lipgloss will automatically wrap any text based on the current dimensions of the user's term.
	s := lipgloss.
		NewStyle().
		MaxWidth(m.width).
		MaxHeight(m.height)

	b.WriteString(m.renderInit())

	// The final "\n" is important as when Bubbletea exists it resets the last
	// line of output, meaning without a new line we'd lose the final line
	return s.Render(b.String() + "\n")
}

func (m *InitModel) renderInit() string {
	b := strings.Builder{}

	switch true {
	case m.Err != nil:
		b.WriteString(colors.Red(fmt.Sprintf("Error: %s", m.Err.Error())).String())
	case m.Status == StatusInitialized:
		b.WriteString(fmt.Sprintf("%s\n\n", colors.Green("Ready to build with Keel!").String()))

		if len(m.GeneratedFiles) > 0 {

			b.WriteString("The following files were generated:\n")
			b.WriteString("===================================\n")

			for _, f := range m.GeneratedFiles {
				b.WriteString(fmt.Sprintf("%s\n", colors.Gray(fmt.Sprintf("- %s", f.Path)).String()))
			}
		}

		b.WriteString("\n")

		b.WriteString(colors.Cyan("Visit https://keel.notaku.site/ to get started.").String())

		b.WriteString("\n")
	}

	return b.String()
}

type InitialisedMsg struct {
	GeneratedFiles codegen.GeneratedFiles
	Err            error
}

func Init(dir string) tea.Cmd {
	return func() tea.Msg {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			err = os.MkdirAll(dir, os.ModePerm)

			if err != nil {
				return InitialisedMsg{
					Err: fmt.Errorf("Could not create the specified directory."),
				}
			}
		}

		empty := isDirEmpty(dir)

		if !empty {
			return InitialisedMsg{
				Err: fmt.Errorf("The directory you are trying to initialise is not empty"),
			}
		}

		files := codegen.GeneratedFiles{}

		files = append(files, &codegen.GeneratedFile{
			Path: ".gitignore",
			Contents: `.build/
node_modules/
.DS_Store
			`,
		})

		files = append(files, &codegen.GeneratedFile{
			Path:     "schema.keel",
			Contents: "// Visit https://keel.notaku.site/ for documentation on how to get started",
		})

		files = append(files, &codegen.GeneratedFile{
			Path: "keelconfig.yaml",
			Contents: `# Visit https://keel.notaku.site/documentation/environment-variables-and-secrets for more
# information about environment variables and secrets
environment:
  default:
  development:
  test:
  staging:
  production:

secrets:
`,
		})

		err := files.Write(dir)

		if err != nil {
			return InitialisedMsg{
				Err: err,
			}
		}

		return InitialisedMsg{
			GeneratedFiles: files,
		}
	}
}
