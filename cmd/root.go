package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fabio42/sasqwatch/ui"
	"github.com/fabio42/sasqwatch/ui/theme"

	tea "charm.land/bubbletea/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rootFlags = struct {
		chgExit  bool
		debug    bool
		diff     bool
		errExit  bool
		permDiff bool
		interval uint
		records  uint
		title    string
	}{}

	rootCmd = &cobra.Command{
		Use:   "sasqwatch [flags] command",
		Short: "sasqwatch",
		Long:  "sasqwatch is a tool to execute a program periodically, showing output fullscreen.",
		Args:  cobra.MinimumNArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setLogger(rootFlags.debug); err != nil {
				return fmt.Errorf("failed to configure logger: %w", err)
			}

			hostname := rootFlags.title
			if hostname == "" {
				var err error
				hostname, err = os.Hostname()
				if err != nil {
					log.Warn().Msgf("could not resolve hostname: %v", err)
					hostname = "(✖╭╮✖)"
				}
			}

			cfg := ui.Config{
				Interval: time.Second * time.Duration(rootFlags.interval),
				History:  int(rootFlags.records),
				HostName: hostname,
				Cmd:      strings.Join(args, " "),
				ChgExit:  rootFlags.chgExit,
				Diff:     rootFlags.diff,
				ErrExit:  rootFlags.errExit,
				PermDiff: rootFlags.permDiff,
				Theme:    theme.DefaultTheme(),
			}

			m := ui.NewModel(cfg)
			if _, err := tea.NewProgram(m).Run(); err != nil {
				return fmt.Errorf("program error: %w", err)
			}
			return nil
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
		log.Fatal().Msgf("error: %v", err)
	}
}
