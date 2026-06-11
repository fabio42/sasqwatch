package ui

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
)

const statusHeight = 2

// statusView generates the status bar rendered as a single-line string.
func (m *Model) statusView() string {
	var left, right, modeData, clip, diff, records, date string
	var cmd cmdData
	t := m.cfg.Theme

	mainStyle := lipgloss.NewStyle().Background(t.StatusBgColor).Foreground(t.StatusFgColor)

	if m.cmdIdx == 0 {
		records = fmt.Sprintf(" latest/%d ", m.cmdRecords)
	} else {
		records = fmt.Sprintf(" %d/%d ", m.cmdIdx+1, m.cmdRecords)
	}
	records = mainStyle.Foreground(t.StatusOptionColor).Render(records)

	var bg = t.StatusRunColor
	if m.paused {
		cmd = m.cmdsData[len(m.cmdsData)-1-m.cmdIdx]
		bg = t.StatusStopColor
		modeData = fmt.Sprintf(" ■ Every %s: %s ", m.cfg.Interval, m.cfg.Cmd)
	} else {
		cmd = m.cmdsData[len(m.cmdsData)-1]
		modeData = fmt.Sprintf(" ▶ Every %s: %s ", m.cfg.Interval, m.cfg.Cmd)
	}

	switch {
	case m.copyCb:
		clip = mainStyle.Foreground(t.StatusRunColor).Render(" Copied!")
	case m.copyErr:
		clip = mainStyle.Foreground(t.StatusStopColor).Render(" Copy failed!")
	}

	if m.diffOption != diffOff {
		var diffMode string
		if m.diffOption == diffSimple {
			diffMode = t.OptionSeparator + "diff "
		} else {
			diffMode = t.OptionSeparator + "permDiff "
		}
		diff = mainStyle.Foreground(t.StatusOptionColor).Render(diffMode)
	}

	// On start the date is unset until the first command execution completes.
	if cmd.date.Equal(time.Time{}) && m.firstRun {
		cmd.date = time.Now()
	}
	date = fmt.Sprintf("%s: %s", m.cfg.HostName, cmd.date.Format("Mon Jan 02 15:04:05 2006"))
	mode := mainStyle.Background(bg).AlignHorizontal(lipgloss.Left).Render(modeData)
	left = mode + records + diff + clip

	left = m.truncStatus(left, len([]rune(date)))

	if len([]rune(left+date)) > m.width {
		date = m.truncStatus(date, 1)
	}

	right = mainStyle.Width(m.width - (lipgloss.Width(left))).AlignHorizontal(lipgloss.Right).Render(date)
	return left + right
}

// truncStatus truncates str so it fits within (m.width - width) columns,
// appending an ellipsis if any characters are dropped.
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
