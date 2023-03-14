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

## Demo
<p align="center">
    <img width="700" src="demo.gif" />
</p>

## Installation

```bash
go install github.com/fabio42/sasqwatch@latest
```

## Usage
```
sasqwatch is a tool to execute a program periodically, showing output fullscreen.

Usage:
  sasqwatch [flags] command

Flags:
  -D, --debug              Enable debug log, out will will be saved in
  -d, --diff               Highlight the differences between successive updates.
  -h, --help               help for sasqwatch
  -n, --interval uint      Specify update interval. (default 2)
  -P, --permdiff           Highlight the differences between successive updates since the first iteration.
  -r, --records uint       Specify how many stdout records are kept in memory. (default 50)
  -T, --set-title string   Replace the hostname in the status bar by a custom string.
  -v, --version            version for sasqwatch
```

## A word on the implementation

Most of the complex problems were solved using the `bubbletea` libraries:

* The command execution ticking system relies on the `timer` module, with precision at the second level. This could be improved, but this aspect has not been extensively tested yet, so you should not rely on this program if you need precise timing.

* The command output handling relies on the `viewport` module. I encountered some limitations with the current version, which [prevent horizontal scrolling](https://github.com/charmbracelet/bubbles/issues/236) and [line wrapping](https://github.com/charmbracelet/bubbles/issues/56). Therefore, a patched version of `viewport` is provided.

I attempted to implement line wrapping but faced challenges, particularly with very long outputs and handling of diffs. Eventually, I came across [this patch](https://github.com/charmbracelet/bubbles/pull/240) provided by @tty2 that is still pending review. The patch is relatively easy to understand and works very well for the use case of `sasqwatch`. As a result, `sasqwatch` currently does not wrap lines, but it allows horizontal scrolling.

Finally, Windows is not supported at the moment, but this should be easy to implement!
