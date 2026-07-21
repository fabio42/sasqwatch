# sasqwatch

Introducing `sasqwatch`, a modern take on the classic watch command using charm.sh libraries.

This is a simple implementation that showcases some of the capabilities of the bubletea libraries. 
It started with a basic idea, but as we discussed it with friends, it grew to become something more. 
The initial goal was to create a `watch` command version that would allow updates to be triggered at any time between the interval.

## Main features

* Provides main features of the original watch command
* Trigger the command manually at any time between intervals
* History feature to go back in time and navigate through recorded outputs
* Records full command output, with vertical and horizontal scrolling support
* Provides the ability to quickly copy command output to your clipboard
* Allows you to set a custom title
* Mouse support for scrolling
* PTY mode (`-t`) for commands that adapt their output to terminal width (tables, colored output, etc.)

## Demo
<p align="center">
    <img width="700" src="demo.gif" />
</p>

## Installation

Ensure you have Go 1.20 or later installed.

```bash
go install github.com/fabio42/sasqwatch@latest
```

Alternatively, you can clone the repository and build from source:

```bash
git clone https://github.com/fabio42/sasqwatch.git
cd sasqwatch
go mod tidy
go build
```

Note: This project requires Go 1.20 or later.

`sasqwatch` is now also available through [tea package manager](https://tea.xyz/):

```bash
sh <(curl tea.xyz) sasqwatch --help
```

## Usage
```
sasqwatch is a tool to execute a program periodically, showing output fullscreen.

Usage:
  sasqwatch [flags] command

Flags:
  -g, --chgexit            Exit when output from command changes
  -D, --debug              Enable debug log
  -d, --diff               Highlight the differences between successive updates
  -e, --errexit            Exit if command has a non-zero exit
  -h, --help               help for sasqwatch
  -n, --interval uint      Specify update interval (default 2)
  -P, --permdiff           Highlight the differences between successive updates since the first iteration
  -t, --pty                Run the command on a pseudo-terminal (TTY) so terminal-aware tools format correctly; may emit raw escape codes for screen-control programs
  -r, --records uint       Specify how many stdout records are kept in memory (default 50)
  -T, --set-title string   Replace the hostname in the status bar by a custom string
  -v, --version            version for sasqwatch
```

## Adjusting the Interval on the Fly

Press `+` (or `=`) to increase the interval and `-` (or `_`) to decrease it while the program is running. The step is magnitude-scaled so it feels natural at any speed: 1s steps below 10s, 5s below 1m, 30s below 5m, 1m below 1h, and 5m above that. The countdown restarts immediately and the interval is floored at 1s. The change also takes effect while paused — the new value applies when you resume.

## Command History

`sasqwatch` keeps track of the command output history. You can use the `[` and `]` keys to travel back in time and visualize previous records. While viewing previous records, `sasqwatch` stops recording and enters `pause` mode. You can activate recording again by pressing the `space` key.

To save memory and control memory footprint, only changing outputs are recorded. In other words, if there are no changes in the stdout between two executions, it won't be recorded. By default, only the last 50 command outputs are recorded, but this can be adjusted using the `-r <value>` option.

## PTY Mode

By default, `sasqwatch` runs the watched command with standard pipes. This is safe and predictable, but some tools detect that their output is not going to a terminal and fall back to a simplified layout — for example, a CLI that normally draws a formatted table will collapse its columns when it sees a pipe.

The `-t` / `--pty` flag runs the command on a pseudo-terminal (PTY) sized to sasqwatch's current display width. This makes `isatty()` return true for the child process, so terminal-aware tools format their output exactly as they would in a regular shell:

```bash
sasqwatch -t -n 5 jira my_sprint
```

**When to use it:** any command that produces width-aware tables, colored output, or otherwise adapts its formatting based on whether stdout is a terminal.

**When to avoid it:** commands that emit cursor-movement or screen-control sequences (`top`, `htop`, ncurses-based tools) will dump raw escape codes into the viewport rather than rendering correctly. Similarly, PTY mode causes tools to emit ANSI color codes, which can make `--diff` / `--permdiff` output noisier since the diff runs over the raw bytes including escape sequences.

## A word on the implementation

Most of the complex problems were solved using the `bubbletea` libraries:

* The command execution ticking system relies on the `timer` module, with precision at the second level. This could be improved, but this aspect has not been extensively tested yet, so you should not rely on this program if you need precise timing.

* The command output handling relies on the `viewport` module. I encountered some limitations with the current version, which [prevent horizontal scrolling](https://github.com/charmbracelet/bubbles/issues/236) and [line wrapping](https://github.com/charmbracelet/bubbles/issues/56). Therefore, a patched version of `viewport` is provided.

I attempted to implement line wrapping but faced challenges, particularly with very long outputs and handling of diffs. Eventually, I came across [this patch](https://github.com/charmbracelet/bubbles/pull/240) provided by @tty2 that is still pending review. The patch is relatively easy to understand and works very well for the use case of `sasqwatch`. As a result, `sasqwatch` currently does not wrap lines, but it allows horizontal scrolling.

Finally, Windows is not supported at the moment, but this should be easy to implement!
