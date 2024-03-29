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
	ChgExit  bool
	Diff     bool
	ErrExit  bool
	PermDiff bool
}

type cmdData struct {
	stdout     []byte
	stdoutDiff string
	exitCode   int
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
	cfg           Config
	execCh        chan cmdData
	cmdPerpDiff   string
	cmdIdx        int
	cmdRecords    int
	diffOption    int
	paused        bool
	copyCb        bool
	forcedRun     bool
	firstRun      bool
	printHelp     bool
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
		viewport:   &vp,
		keymap:     km,
		paused:     false,
		firstRun:   true,
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
			if m.paused {
				cmds = append(cmds, m.timer.Stop())
			} else {
				m.cmdIdx = 0
				cmds = append(cmds, runCmdEvent)
			}
		case key.Matches(msg, m.keymap.run):
			m.forcedRun = true
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
				log.Debug().Str("function", "ModelUpdate").Str("Case", "KeyMsg").Str("Key", "Prev").Msgf("cmdIdx: %v - cmdRecords: %v", m.cmdIdx, m.cmdRecords)
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
				log.Debug().Str("function", "ModelUpdate").Str("Case", "KeyMsg").Str("Key", "copy").Msgf("Clipboard err: %v", err)
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
		log.Debug().Str("function", "ModelUpdate").Str("Case", "TimeoutMsg").Msg("")
		cmds = append(cmds, runCmdEvent)

	case runCmd:
		if !inProgress {
			log.Debug().Str("function", "ModelUpdate").Str("Case", "runCmd").Msg("Trigger command")
			go execCmd(strings.Fields(m.cfg.Cmd), m.execCh)
		} else {
			log.Debug().Str("function", "ModelUpdate").Str("Case", "runCmd").Msg("Skipping command")
		}
		cmds = append(cmds, waitCmd(m.execCh))

	case cmdData:
		log.Debug().Str("function", "ModelUpdate").Str("Case", "cmdData").Int("timerId", m.timer.ID()).Bool("Paused", m.paused).Bool("timerRunning", m.timer.Running()).Msg("")

		if !m.paused {
			m.timer = timer.NewWithInterval(m.cfg.Interval, time.Second)
			cmds = append(cmds, m.timer.Init())
			log.Debug().Str("function", "ModelUpdate").Str("Case", "cmdData").Int("timerId", m.timer.ID()).Bool("timerRunning", m.timer.Running()).Msg("New timer")
		} else if m.forcedRun {
			m.forcedRun = false
		}
		t := m.procCmdData(msg)
		if t != nil {
			return m, t
		}
		cmds = append(cmds, updateSdtOutEvent)
		if m.firstRun {
			log.Debug().Str("function", "ModelUpdate").Str("Case", "cmdData").Bool("firstRun", m.firstRun).Msg("")
			m.firstRun = false
		}

	case updateSdtOut:
		log.Debug().Str("function", "ModelUpdate").Str("Case", "updateSdtOut").Msg("Event received")
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

// procCmdData update in memory command execution data array
func (m *Model) procCmdData(d cmdData) tea.Cmd {
	if m.cfg.ErrExit && d.exitCode != 0 {
		log.Debug().Str("function", "procCmdData").Any("cmdErr", d.exitCode).Msg("Exiting")
		return tea.Quit
	}

	if string(d.stdout) != string(m.cmdsData[len(m.cmdsData)-1].stdout) {
		if m.cfg.ChgExit && !m.firstRun {
			log.Debug().Str("function", "procCmdData").Bool("optChgExit", m.cfg.ChgExit).Bool("FirstRun", m.firstRun).Msg("Exiting")
			return tea.Quit
		}
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
	return nil
}

// diffStdout process stdout command data and identify diffs
func (m *Model) diffStdout() string {
	log.Debug().Str("function", "ModeldiffStdout").Int("cmdIdx", m.cmdIdx).Int("cmdRecords", m.cmdRecords).Int("History", m.cfg.History).Msg("Diff processing")

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
	log.Debug().Str("function", "execCmd").Msg("")

	inProgress = true
	var cmd cmdData
	var err error

	exeCmd := exec.Command("sh", "-c", strings.Join(command, " "))
	cmd.stdout, err = exeCmd.CombinedOutput()
	if exitError, ok := err.(*exec.ExitError); ok {
		log.Debug().Str("function", "execCmd").Int("ExitCode", exitError.ExitCode()).Msg("")
		cmd.exitCode = exitError.ExitCode()
	}
	cmd.date = time.Now()
	inProgress = false
	outputChan <- cmd
}
