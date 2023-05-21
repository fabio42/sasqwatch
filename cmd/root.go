package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fabio42/sasqwatch/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rootFlags = struct {
		chgExit   bool
		debug     bool
		diff      bool
		errExit   bool
		permDiff  bool
		statusBar bool
		interval  uint
		records   uint
		title     string
	}{}

	rootCmd = &cobra.Command{
		Use:   "sasqwatch [flags] command",
		Short: "sasqwatch",
		Long:  "sasqwatch is a tool to execute a program periodically, showing output fullscreen.",

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(0)
			}

			err := setLogger(rootFlags.debug)
			if err != nil {
				log.Fatal().Msgf("Error failed to configure logger:", err)
			}

			cfg := ui.Config{
				Interval: time.Second * time.Duration(rootFlags.interval),
				History:  int(rootFlags.records),
				HostName: rootFlags.title,
				Cmd:      strings.Join(args, " "),
				ChgExit:  rootFlags.chgExit,
				Diff:     rootFlags.diff,
				ErrExit:  rootFlags.errExit,
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
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.chgExit, "chgexit", "g", false, "Exit when output from command changes")
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.debug, "debug", "D", false, "Enable debug log")
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.diff, "diff", "d", false, "Highlight the differences between successive updates")
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.errExit, "errexit", "e", false, "Exit if command has a non-zero exit")
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.permDiff, "permdiff", "P", false, "Highlight the differences between successive updates since the first iteration")
	rootCmd.PersistentFlags().UintVarP(&rootFlags.interval, "interval", "n", 2, "Specify update interval")
	rootCmd.PersistentFlags().UintVarP(&rootFlags.records, "records", "r", 50, "Specify how many stdout records are kept in memory")
	rootCmd.PersistentFlags().StringVarP(&rootFlags.title, "set-title", "T", "", "Replace the hostname in the status bar by a custom string")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Msgf("Whoops. There was an error while executing your CLI '%s'", err)
	}
}
