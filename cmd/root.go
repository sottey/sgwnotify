package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"sgwnotify/internal/sgwnotify"

	"github.com/spf13/cobra"
)

const defaultLookaheadMinutes = 120
const defaultHTTPTimeout = 15 * time.Second

type cliOptions struct {
	configPath       string
	token            string
	lookaheadMinutes int
	verbose          bool
	outputJSON       bool
	outputPlain      bool
}

func Execute() error {
	opts := cliOptions{}

	rootCmd := &cobra.Command{
		Use:           "sgwnotify",
		Short:         "Notify when ShopGoodwill favorites are ending soon",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return sgwnotify.Run(sgwnotify.Options{
				ConfigPath:              opts.configPath,
				Token:                   opts.token,
				TokenChanged:            cmd.Flags().Changed("token"),
				LookaheadMinutes:        opts.lookaheadMinutes,
				LookaheadMinutesChanged: cmd.Flags().Changed("lookahead-minutes"),
				DefaultLookaheadMinutes: defaultLookaheadMinutes,
				DefaultOpenURL:          sgwnotify.DefaultOpenURL,
				DefaultHTTPTimeout:      defaultHTTPTimeout,
				Verbose:                 opts.verbose,
				Output:                  cmd.OutOrStdout(),
			})
		},
	}

	rootCmd.PersistentFlags().StringVar(&opts.configPath, "config", defaultConfigPath(), "config file path")
	rootCmd.Flags().StringVar(&opts.token, "token", "", "ShopGoodwill bearer token")
	rootCmd.Flags().IntVar(&opts.lookaheadMinutes, "lookahead-minutes", 0, "auction ending lookahead window in minutes")
	rootCmd.Flags().BoolVar(&opts.verbose, "verbose", false, "print run details")

	rootCmd.AddCommand(testNotificationCommand(&opts))
	rootCmd.AddCommand(checkTokenCommand(&opts))
	rootCmd.AddCommand(listCommand(&opts))
	rootCmd.AddCommand(configCommand(&opts))

	return rootCmd.Execute()
}

func checkTokenCommand(opts *cliOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check-token",
		Short: "Check ShopGoodwill token access",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sgwnotify.CheckToken(sgwnotify.Options{
				ConfigPath:              opts.configPath,
				Token:                   opts.token,
				TokenChanged:            cmd.Flags().Changed("token"),
				LookaheadMinutes:        opts.lookaheadMinutes,
				LookaheadMinutesChanged: false,
				DefaultLookaheadMinutes: defaultLookaheadMinutes,
				DefaultOpenURL:          sgwnotify.DefaultOpenURL,
				DefaultHTTPTimeout:      defaultHTTPTimeout,
			}, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.token, "token", "", "ShopGoodwill bearer token")
	return cmd
}

func listCommand(opts *cliOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List favorites ending within the lookahead window",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputMode := sgwnotify.OutputPlain
			if opts.outputJSON && opts.outputPlain {
				return fmt.Errorf("--json and --plain cannot be used together")
			}
			if opts.outputJSON {
				outputMode = sgwnotify.OutputJSON
			}
			return sgwnotify.ListEndingFavorites(sgwnotify.Options{
				ConfigPath:              opts.configPath,
				Token:                   opts.token,
				TokenChanged:            cmd.Flags().Changed("token"),
				LookaheadMinutes:        opts.lookaheadMinutes,
				LookaheadMinutesChanged: cmd.Flags().Changed("lookahead-minutes"),
				DefaultLookaheadMinutes: defaultLookaheadMinutes,
				DefaultOpenURL:          sgwnotify.DefaultOpenURL,
				DefaultHTTPTimeout:      defaultHTTPTimeout,
			}, sgwnotify.ListOptions{
				OutputMode: outputMode,
			}, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.token, "token", "", "ShopGoodwill bearer token")
	cmd.Flags().IntVar(&opts.lookaheadMinutes, "lookahead-minutes", 0, "auction ending lookahead window in minutes")
	cmd.Flags().BoolVar(&opts.outputJSON, "json", false, "print JSON output")
	cmd.Flags().BoolVar(&opts.outputPlain, "plain", false, "print plain text output")
	return cmd
}

func testNotificationCommand(opts *cliOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "test-notification",
		Short: "Send a test notification",
		RunE: func(cmd *cobra.Command, args []string) error {
			openURL, err := sgwnotify.ConfigOpenURL(opts.configPath, sgwnotify.DefaultOpenURL)
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}
			return sgwnotify.TestNotification(openURL)
		},
	}
}

func configCommand(opts *cliOptions) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage sgwnotify config",
	}

	configCmd.AddCommand(&cobra.Command{
		Use:   "set-keywords <keyword>...",
		Short: "Save keywords for newly listed items",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sgwnotify.SaveConfigKeywords(opts.configPath, args)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "clear-keywords",
		Short: "Disable newly listed item keyword monitoring",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sgwnotify.SaveConfigKeywords(opts.configPath, nil)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "set-token <token>",
		Short: "Save the ShopGoodwill bearer token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sgwnotify.SaveConfigToken(opts.configPath, args[0])
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show sgwnotify config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sgwnotify.ShowConfig(opts.configPath, defaultLookaheadMinutes, sgwnotify.DefaultOpenURL, defaultHTTPTimeout, cmd.OutOrStdout())
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Show sgwnotify config path",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), opts.configPath)
			return nil
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "set-lookahead-minutes <minutes>",
		Short: "Save the auction ending lookahead window in minutes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			minutes, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("lookahead minutes must be an integer")
			}
			if minutes <= 0 {
				return fmt.Errorf("lookahead minutes must be greater than 0")
			}
			return sgwnotify.SaveConfigLookaheadMinutes(opts.configPath, minutes)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "set-http-timeout-seconds <seconds>",
		Short: "Save the favorites API HTTP timeout in seconds",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			seconds, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("HTTP timeout seconds must be an integer")
			}
			if seconds <= 0 {
				return fmt.Errorf("HTTP timeout seconds must be greater than 0")
			}
			return sgwnotify.SaveConfigHTTPTimeoutSeconds(opts.configPath, seconds)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate sgwnotify config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := sgwnotify.ValidateConfig(opts.configPath); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Config is valid.")
			return nil
		},
	})

	return configCmd
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, ".config", "sgwnotify", "config.json")
}
