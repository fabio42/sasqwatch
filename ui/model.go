package ui

import (
	"os/exec"
	"strings"
	"time"

	"github.com/fabio42/sasqwatch/ui/theme"
	"github.com/fabio42/sasqwatch/viewport"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/timer"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog/log"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	diffOff = iota
	diffSimple
	diffPerpetual
)

// CommandRunner abstracts shell execution so the model can be tested without spawning processes.
type CommandRunner interface {
	Run(command string) (stdout []byte, exitCode int)
}

// shellRunner is the production implementation of CommandRunner.
type shellRunner struct{}

func (shellRunner) Run(command string) ([]byte, int) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.CombinedOutput()
	if exitErr, ok := err.(*exec.ExitError); ok {
		return out, exitErr.ExitCode()
	}
	return out, 0
}

// Clipboard abstracts clipboard writes so the model can be tested without touching the system clipboard.
type Clipboard interface {
	Write(s string) error
}

// atottoClipboard is the production Clipboard implementation.
type atottoClipboard struct{}

func (atottoClipboard) Write(s string) error {
	// Import kept local so the default build tag for clipboard still works.
	return clipboardWriteAll(s)
}

// Config holds all configuration that cmd passes into the UI model.
type Config struct {
	Interval time.Duration
	History  int
	HostName string // pre-resolved; empty falls back to os.Hostname inside NewModel
	Cmd      string
	ChgExit  bool
	Diff     bool
	ErrExit  bool
	PermDiff bool
	Theme    theme.SasqTheme
	Runner   CommandRunner // optional; defaults to shellRunner{}
	Clip     Clipboard     // optional; defaults to atottoClipboard{}
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
	timer       timer.Model
	viewport    *viewport.Model
	help        help.Model
	keymap      keymap
	cmdsData    []cmdData
	cfg         Config
	execCh      chan cmdData
	cmdPerpDiff string
	cmdIdx      int
	cmdRecords  int
	diffOption  int
	paused      bool
	copyCb      bool
	copyErr     bool // true when the last copy attempt failed
	inProgress  bool // true while a command goroutine is running
	forcedRun   bool
	firstRun    bool
	printHelp   bool
	diffColors  int
	width       int
	height      int
}

type runCmd struct{}
type updateStdOut struct{}
type clipboardNotification struct{}

func runCmdEvent() tea.Msg       { return runCmd{} }
func updateStdOutEvent() tea.Msg { return updateStdOut{} }

func NewModel(cfg Config) Model {
	vp := viewport.New(200, 10)
	vp.MouseWheelEnabled = true

	if cfg.Runner == nil {
		cfg.Runner = shellRunner{}
	}
	if cfg.Clip == nil {
		cfg.Clip = atottoClipboard{}
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
	return runCmdEvent
}

// viewportHeight returns the correct viewport height given the current help state.
func (m Model) viewportHeight() int {
	if m.printHelp {
		return m.height - statusHeight - helpFullHeight
	}
	return m.height - statusHeight - helpHeight
}


func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = m.viewportHeight()
		cmds = append(cmds, updateStdOutEvent)

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.pause):
			if m.paused {
				m.paused = false
				m.cmdIdx = 0
				cmds = append(cmds, runCmdEvent)
			} else {
				m.paused = true
				cmds = append(cmds, m.timer.Stop())
			}
		case key.Matches(msg, m.keymap.run):
			m.forcedRun = true
			if m.paused {
				m.cmdIdx = 0
			}
			return m, runCmdEvent
		case key.Matches(msg, m.keymap.prev):
			if !m.paused {
				m.paused = true
				cmds = append(cmds, m.timer.Stop())
			}
			if m.cmdIdx < m.cmdRecords-1 {
				m.cmdIdx++
				log.Debug().Str("function", "Update").Str("case", "prev").
					Msgf("cmdIdx: %v - cmdRecords: %v", m.cmdIdx, m.cmdRecords)
				cmds = append(cmds, updateStdOutEvent)
			}
		case key.Matches(msg, m.keymap.next):
			if m.cmdIdx > 0 {
				m.cmdIdx--
				cmds = append(cmds, updateStdOutEvent)
			}
		case key.Matches(msg, m.keymap.diff):
			if m.diffOption >= diffPerpetual {
				m.diffOption = diffOff
			} else {
				m.diffOption++
			}
		case key.Matches(msg, m.keymap.copy):
			err := m.cfg.Clip.Write(string(m.cmdsData[len(m.cmdsData)-1-m.cmdIdx].stdout))
			if err != nil {
				log.Debug().Str("function", "Update").Str("case", "copy").
					Msgf("clipboard error: %v", err)
				m.copyErr = true
			} else {
				m.copyCb = true
			}
			return m, tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
				return clipboardNotification{}
			})
		case key.Matches(msg, m.keymap.help):
			m.printHelp = !m.printHelp
			m.viewport.Height = m.viewportHeight()
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
		log.Debug().Str("function", "Update").Str("case", "timeout").Msg("")
		cmds = append(cmds, runCmdEvent)

	case runCmd:
		if !m.inProgress {
			log.Debug().Str("function", "Update").Str("case", "runCmd").Msg("trigger command")
			m.inProgress = true
			go execCmd(m.cfg.Cmd, m.execCh, m.cfg.Runner)
		} else {
			log.Debug().Str("function", "Update").Str("case", "runCmd").Msg("command already in progress, skipping")
		}
		cmds = append(cmds, waitCmd(m.execCh))

	case cmdData:
		m.inProgress = false
		log.Debug().Str("function", "Update").Str("case", "cmdData").
			Int("timerId", m.timer.ID()).Bool("paused", m.paused).Bool("timerRunning", m.timer.Running()).Msg("")

		if !m.paused {
			m.timer = timer.New(m.cfg.Interval, timer.WithInterval(time.Second))
			cmds = append(cmds, m.timer.Init())
			log.Debug().Str("function", "Update").Str("case", "cmdData").
				Int("timerId", m.timer.ID()).Bool("timerRunning", m.timer.Running()).Msg("new timer started")
		} else if m.forcedRun {
			m.forcedRun = false
		}
		if t := m.procCmdData(msg); t != nil {
			return m, t
		}
		cmds = append(cmds, updateStdOutEvent)
		if m.firstRun {
			log.Debug().Str("function", "Update").Str("case", "cmdData").Bool("firstRun", m.firstRun).Msg("")
			m.firstRun = false
		}

	case updateStdOut:
		log.Debug().Str("function", "Update").Str("case", "updateStdOut").Msg("event received")
		if m.diffOption != diffOff {
			m.viewport.SetContent(m.renderDiff())
		} else {
			m.viewport.SetContent(string(m.cmdsData[len(m.cmdsData)-1-m.cmdIdx].stdout))
		}

	case clipboardNotification:
		m.copyCb = false
		m.copyErr = false
	}

	var cmd tea.Cmd
	*m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	var str strings.Builder
	str.WriteString(m.statusView())
	str.WriteString("\n\n")
	str.WriteString(m.viewport.View())
	if m.printHelp {
		str.WriteString(m.helpFullView())
	} else {
		str.WriteString(m.helpView())
	}
	v := tea.NewView(str.String())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// procCmdData updates the in-memory command history ring buffer.
// Returns a non-nil tea.Cmd only when a forced exit condition is met.
func (m *Model) procCmdData(d cmdData) tea.Cmd {
	if m.cfg.ErrExit && d.exitCode != 0 {
		log.Debug().Str("function", "procCmdData").Int("exitCode", d.exitCode).Msg("errExit: quitting")
		return tea.Quit
	}

	if string(d.stdout) != string(m.cmdsData[len(m.cmdsData)-1].stdout) {
		if m.cfg.ChgExit && !m.firstRun {
			log.Debug().Str("function", "procCmdData").Msg("chgExit: output changed, quitting")
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
		// Output unchanged; only refresh the timestamp.
		m.cmdsData[len(m.cmdsData)-1].date = d.date
	}
	return nil
}

// diffSegment represents a single equal or inserted span from a diff.
type diffSegment struct {
	text     string
	inserted bool
}

// computeDiff returns the diff segments between two strings and, if in perpetual-diff
// mode, the updated perpetual baseline. This function is pure and contains no rendering.
func computeDiff(before, current, perpBase string, perpetual bool) (segments []diffSegment, newPerpBase string) {
	dmp := diffmatchpatch.New()

	var from string
	if perpetual {
		from = perpBase
	} else {
		from = before
	}

	diffs := dmp.DiffMain(from, current, false)

	var allDiff strings.Builder
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			segments = append(segments, diffSegment{text: d.Text, inserted: false})
			if perpetual {
				allDiff.WriteString(d.Text)
			}
		case diffmatchpatch.DiffInsert:
			segments = append(segments, diffSegment{text: d.Text, inserted: true})
			if perpetual {
				// Tag every rune that has changed by toggling between two smiley sentinels.
				// This is an acknowledged approximation; see model.go for context.
				for _, ch := range d.Text {
					if ch == '☺' {
						allDiff.WriteRune('☻')
					} else {
						allDiff.WriteRune('☺')
					}
				}
			}
		}
	}

	if perpetual {
		newPerpBase = allDiff.String()
	}
	return segments, newPerpBase
}

// renderDiff computes the diff and applies lipgloss styling to insertions.
func (m *Model) renderDiff() string {
	log.Debug().Str("function", "renderDiff").
		Int("cmdIdx", m.cmdIdx).Int("cmdRecords", m.cmdRecords).Int("history", m.cfg.History).
		Msg("diff processing")

	if m.diffOption > diffOff && m.cmdPerpDiff == "" {
		m.cmdPerpDiff = string(m.cmdsData[len(m.cmdsData)-1].stdout)
	}
	if m.cmdRecords == 0 || (m.cmdRecords == m.cfg.History && m.cmdIdx == m.cfg.History-1) {
		return string(m.cmdsData[len(m.cmdsData)-1].stdout)
	}

	current := string(m.cmdsData[len(m.cmdsData)-1-m.cmdIdx].stdout)
	before := string(m.cmdsData[len(m.cmdsData)-2-m.cmdIdx].stdout)

	segments, newPerpBase := computeDiff(before, current, m.cmdPerpDiff, m.diffOption == diffPerpetual)
	if m.diffOption == diffPerpetual {
		m.cmdPerpDiff = newPerpBase
	}

	insertStyle := lipgloss.NewStyle().Background(m.cfg.Theme.DiffColor)
	var out strings.Builder
	for _, seg := range segments {
		if seg.inserted {
			out.WriteString(insertStyle.Render(seg.text))
		} else {
			out.WriteString(seg.text)
		}
	}
	return out.String()
}

// waitCmd bridges a cmdData channel result into the tea.Msg stream.
func waitCmd(resp chan cmdData) tea.Cmd {
	return func() tea.Msg {
		return cmdData(<-resp)
	}
}

// execCmd runs command via the provided CommandRunner and sends the result to outputChan.
// This function is meant to be called as a goroutine. It does not touch any model state.
func execCmd(command string, outputChan chan<- cmdData, runner CommandRunner) {
	log.Debug().Str("function", "execCmd").Msg("")
	stdout, exitCode := runner.Run(command)
	outputChan <- cmdData{
		stdout:   stdout,
		exitCode: exitCode,
		date:     time.Now(),
	}
}
