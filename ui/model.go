package ui

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fabio42/sasqwatch/viewport"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	diffOff = iota
	diffSimple
	diffPerpetual
)

var (
	inProgress bool
)

type Config struct {
	Interval time.Duration
	History  int
	HostName string
	Cmd      string
	Diff     bool
	PermDiff bool
}

type cmdData struct {
	stdout     []byte
	stdoutDiff string
	err        error
	date       time.Time
	header     string
}

type cmdQuery struct {
	cmd    []string
	result chan cmdData
}

type Model struct {
	timer         timer.Model
	viewport      *viewport.Model
	help          help.Model
	keymap        keymap
	cmdsData      []cmdData
	cmdFields     []string
	cmdPerpDiff   string
	cmdIdx        int
	cmdRecords    int
	cmdError      bool
	execCh        chan cmdData
	diffOption    int
	paused        bool
	foredRun      bool
	copyCb        bool
	printHelp     bool
	cfg           Config
	diffColors    int
	width, height int
}

type runCmd struct{}
type updateSdtOut struct{}
type clipboardNotification struct{}

func runCmdEvent() tea.Msg       { return runCmd{} }
func updateSdtOutEvent() tea.Msg { return updateSdtOut{} }

func NewModel(cfg Config) Model {
	vp := viewport.New(200, 10)
	vp.MouseWheelEnabled = true

	t := timer.NewWithInterval(cfg.Interval, time.Second)

	if cfg.HostName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "(✖╭╮✖)"
		}
		cfg.HostName = hostname
	}

	diffOpt := 0
	if cfg.Diff {
		diffOpt = 1
	} else if cfg.PermDiff {
		diffOpt = 2
	}

	return Model{
		cfg:        cfg,
		timer:      t,
		viewport:   &vp,
		keymap:     km,
		paused:     false,
		cmdFields:  strings.Fields(cfg.Cmd),
		cmdsData:   make([]cmdData, cfg.History),
		execCh:     make(chan cmdData),
		diffColors: 1,
		diffOption: diffOpt,
		help:       help.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.timer.Init(),
		func() tea.Msg { return runCmd{} },
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		if m.printHelp {
			m.viewport.Height = msg.Height - statusHeight - helpFullHeight
		} else {
			m.viewport.Height = msg.Height - statusHeight - helpHeight
		}
		cmds = append(cmds, updateSdtOutEvent)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.pause):
			m.paused = !m.paused
			if !m.paused {
				m.cmdIdx = 0
			}
			return m, m.timer.Toggle()
		case key.Matches(msg, m.keymap.run):
			m.foredRun = true
			if m.paused {
				m.cmdIdx = 0
			}
			return m, runCmdEvent
		case key.Matches(msg, m.keymap.prev):
			var rep tea.Cmd
			if !m.paused {
				m.paused = !m.paused
				rep = m.timer.Stop()
			}
			if m.cmdIdx < m.cmdRecords-1 {
				m.cmdIdx++
				log.Debug().Msgf("cmdIdx: %v - cmdRecords: %v", m.cmdIdx, m.cmdRecords)
				cmds = append(cmds, updateSdtOutEvent)
			}
			cmds = append(cmds, rep)
		case key.Matches(msg, m.keymap.next):
			if m.cmdIdx > 0 {
				m.cmdIdx--
				cmds = append(cmds, updateSdtOutEvent)
			}
		case key.Matches(msg, m.keymap.diff):
			if m.diffOption >= diffPerpetual {
				m.diffOption = diffOff
			} else {
				m.diffOption++
			}
		case key.Matches(msg, m.keymap.copy):
			err := clipboard.WriteAll(string(m.cmdsData[len(m.cmdsData)-1-m.cmdIdx].stdout))
			if err != nil {
				log.Debug().Msgf("Clipboard err: %v", err)
			}
			m.copyCb = true
			// Provide visual feedback that copy to clipboard was successful
			return m, tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
				return clipboardNotification{}
			})
		case key.Matches(msg, m.keymap.help):
			m.printHelp = !m.printHelp
			if m.printHelp {
				m.viewport.Height = m.height - 6
			} else {
				m.viewport.Height = m.height - 3
			}
		}
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	case timer.StartStopMsg:
		var cmd tea.Cmd
		if !m.paused {
			m.timer.Timeout = m.cfg.Interval
		}
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	case timer.TimeoutMsg:
		log.Debug().Msg("TimeoutMsg")
		cmds = append(cmds, runCmdEvent)

	case runCmd:
		if !inProgress {
			// lock required ?
			log.Debug().Msg("runCmd -- Exec command")
			go execCmd(m.cmdFields, m.execCh)
		} else {
			log.Debug().Msg("runCmd -- Skipping already in progress")
		}
		cmds = append(cmds, waitCmd(m.execCh))

	case cmdData:
		log.Debug().Msg("cmdData -- incoming data")
		if !m.paused {
			m.timer.Timeout = m.cfg.Interval
			cmds = append(cmds, m.timer.Start())
			m.procCmdData(msg)
			cmds = append(cmds, updateSdtOutEvent)
		} else if m.foredRun {
			m.foredRun = false
			m.procCmdData(msg)
			cmds = append(cmds, updateSdtOutEvent)
		}

	case updateSdtOut:
		log.Debug().Msg("Update Stdout")
		if m.diffOption != diffOff {
			m.viewport.SetContent(m.diffStdout())
		} else {
			m.viewport.SetContent(string(m.cmdsData[len(m.cmdsData)-1-m.cmdIdx].stdout))
		}

	case clipboardNotification:
		m.copyCb = false
	}

	var cmd tea.Cmd
	*m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var str strings.Builder
	str.WriteString(m.statusView())
	str.WriteString("\n\n")
	str.WriteString(m.viewport.View())
	if m.printHelp {
		str.WriteString(m.helpFullView())
	} else {
		str.WriteString(m.helpView())
	}
	return str.String()
}

// procCmdData update in memory command command execution data array
func (m *Model) procCmdData(d cmdData) {
	if string(d.stdout) != string(m.cmdsData[len(m.cmdsData)-1].stdout) {
		b := make([]cmdData, cap(m.cmdsData))
		copy(b, m.cmdsData[1:])
		b[len(b)-1] = d
		m.cmdsData = b
		if m.cmdRecords < m.cfg.History {
			m.cmdRecords++
		}
	} else {
		m.cmdsData[len(m.cmdsData)-1].date = d.date
	}
}

// diffStdout process stdout command data and identify diffs
func (m *Model) diffStdout() string {
	log.Debug().Msgf("cmdIdx: %v - cmdRecords: %v - history: %v", m.cmdIdx, m.cmdRecords, m.cfg.History)
	if m.diffOption > diffOff && m.cmdPerpDiff == "" {
		m.cmdPerpDiff = string(m.cmdsData[len(m.cmdsData)-1].stdout)
	}
	if m.cmdRecords == 0 || (m.cmdRecords == m.cfg.History && m.cmdIdx == m.cfg.History-1) {
		return string(m.cmdsData[len(m.cmdsData)-1].stdout)
	}

	var str, allDiffStdOut strings.Builder
	var diffs []diffmatchpatch.Diff
	dmp := diffmatchpatch.New()
	current := m.cmdsData[len(m.cmdsData)-1-m.cmdIdx]
	before := m.cmdsData[len(m.cmdsData)-2-m.cmdIdx]

	if m.diffOption == diffPerpetual {
		diffs = dmp.DiffMain(m.cmdPerpDiff, string(current.stdout), false)
	} else {
		diffs = dmp.DiffMain(string(before.stdout), string(current.stdout), false)
	}

	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			str.WriteString(d.Text)
			if m.diffOption == diffPerpetual {
				allDiffStdOut.WriteString(d.Text)
			}
		case diffmatchpatch.DiffInsert:
			str.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("1")).Render(d.Text))
			if m.diffOption == diffPerpetual {
				// This is a very quick take on this, this requires some research but can certainly be improved.
				// Here we just maintain a string tagging all runes that have changed with a smiley.
				for _, ch := range d.Text {
					if ch == '☺' {
						allDiffStdOut.WriteString("☻")
					} else {
						allDiffStdOut.WriteString("☺")
					}
				}
			}
		}
	}
	if m.diffOption == diffPerpetual {
		m.cmdPerpDiff = allDiffStdOut.String()
	}
	return str.String()
}

// waitCmd intercept cmdData and inject it in tea model
func waitCmd(resp chan cmdData) tea.Cmd {
	return func() tea.Msg {
		return cmdData(<-resp)
	}
}

// execCmd execute a shell command
// This function is meant to be run as a goroutine
func execCmd(command []string, outputChan chan<- cmdData) {
	log.Debug().Msg("Goroutine exec")
	inProgress = true
	var cmd cmdData
	exeCmd := exec.Command("sh", "-c", strings.Join(command, " "))
	cmd.stdout, cmd.err = exeCmd.CombinedOutput()
	cmd.date = time.Now()
	inProgress = false
	outputChan <- cmd
}
