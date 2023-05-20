package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const statusHeight = 2

// statusView generate status bar output
func (m *Model) statusView() string {
	var left, right, modeData, clip, diff, records, date string
	var cmd cmdData
	var bg lipgloss.Color

	runColor := lipgloss.Color("2")
	stopColor := lipgloss.Color("1")
	notifColor := lipgloss.Color("3")
	mainStyle := lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("7"))

	if m.cmdIdx == 0 {
		records = fmt.Sprintf(" latest/%d ", m.cmdRecords)
	} else {
		records = fmt.Sprintf(" %d/%d ", m.cmdIdx+1, m.cmdRecords)
	}
	records = mainStyle.Copy().Foreground(notifColor).Render(records)

	if m.paused {
		cmd = m.cmdsData[len(m.cmdsData)-1-m.cmdIdx]
		bg = stopColor
		modeData = fmt.Sprintf(" ■ Every %s: %s ", m.cfg.Interval, m.cfg.Cmd)
	} else {
		cmd = m.cmdsData[len(m.cmdsData)-1]
		bg = runColor
		modeData = fmt.Sprintf(" ▶ Every %s: %s ", m.cfg.Interval, m.cfg.Cmd)
	}
	if m.copyCb {
		clip = mainStyle.Copy().Foreground(runColor).Render(" Copied!")
	}

	if m.diffOption != diffOff {
		var diffMode string
		if m.diffOption == diffSimple {
			diffMode = "| diff "
		} else {
			diffMode = "| permDiff "
		}
		diff = mainStyle.Copy().Foreground(notifColor).Render(diffMode)
	}

	// On start date is not set until first command execution is done
	if cmd.date.Equal(time.Time{}) && m.firstRun {
		m.firstRun = false
		cmd.date = time.Now()
	}
	date = fmt.Sprintf("%s: %s", m.cfg.HostName, cmd.date.Format("Mon Jan 02 15:04:05 2006"))
	mode := mainStyle.Copy().Background(bg).AlignHorizontal(lipgloss.Left).Render(modeData)
	left = mode + records + diff + clip

	left = m.truncStatus(left, len([]rune(date)))

	if len([]rune(left+date)) > m.width {
		date = m.truncStatus(date, 1)
	}

	right = mainStyle.Copy().Width(m.width - (lipgloss.Width(left))).AlignHorizontal(lipgloss.Right).Render(date)
	return left + right
}

// truncStatus truncate status line when screen is too small
func (m *Model) truncStatus(str string, width int) string {
	runes := []rune(str)
	initialLenStr := len(runes)
	statusSize := m.width - width
	for statusSize-lipgloss.Width(string(runes)) < 1 {
		if i := len(runes) - 1; i > 0 {
			runes = runes[:i]
		} else {
			break
		}
	}
	if initialLenStr > len(runes) {
		runes[len(runes)-1] = '…'
	}
	return string(runes)
}
