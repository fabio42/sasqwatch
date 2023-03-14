package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"sasqwatch/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var Version = "dev"

var (
	rootFlags = struct {
		diff      bool
		permDiff  bool
		statusBar bool
		debug     bool
		interval  uint
		records   uint
		title     string
	}{}

	rootCmd = &cobra.Command{
		Use:     "sasqwatch [flags] command",
		Version: Version,
		Short:   "sasqwatch",
		Long:    "sasqwatch is a tool to execute a program periodically, showing output fullscreen.",

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(0)
			}

			cfg := ui.Config{
				Interval: time.Second * time.Duration(rootFlags.interval),
				History:  int(rootFlags.records),
				HostName: rootFlags.title,
				Cmd:      strings.Join(args, " "),
				Diff:     rootFlags.diff,
				PermDiff: rootFlags.permDiff,
			}

			m := ui.NewModel(cfg)
			if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
				fmt.Println("Uh oh, we encountered an error:", err)
				os.Exit(1)
			}
		},
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.diff, "diff", "d", false, "Highlight the differences between successive updates.")
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.permDiff, "permdiff", "P", false, "Highlight the differences between successive updates since the first iteration.")
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.debug, "debug", "D", false, "Enable debug log, out will will be saved in ") // XXX
	rootCmd.PersistentFlags().UintVarP(&rootFlags.interval, "interval", "n", 2, "Specify update interval.")
	rootCmd.PersistentFlags().UintVarP(&rootFlags.records, "records", "r", 50, "Specify how many stdout records are kept in memory.")
	rootCmd.PersistentFlags().StringVarP(&rootFlags.title, "set-title", "T", "", "Replace the hostname in the status bar by a custom string.")

	err := setLogger(rootFlags.debug)
	if err != nil {
		log.Fatal().Msgf("Error failed to configure logger:", err)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Msgf("Whoops. There was an error while executing your CLI '%s'", err)
	}
}
