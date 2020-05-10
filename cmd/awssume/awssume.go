package main

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"
	"text/tabwriter"

	"github.com/gkze/awssume/pkg/awssume"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// errCurrentUser is returned when the current used cannot be determined
const errCurrentUser string = "error determining current user: %w"

var (
	// errTooFewArguments is returned when there are not enough arguments passed
	errTooFewArguments error = errors.New("not enough arguments provided")

	// Version is dynamically injected at build time
	Version string
)

func main() {
	rootCmd := cobra.Command{
		Use:   "awssume [command]",
		Short: "CLI for performing sts:AssumeRole",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return nil
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display awssume version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("awssume version %s\n", Version)
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l", "ls"},
		Short:   "List configured Roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			tw := tabwriter.NewWriter(os.Stdout, 0, 8, 4, ' ', 0)

			curUser, err := user.Current()
			if err != nil {
				return fmt.Errorf(errCurrentUser, err)
			}

			cfg, err := awssume.NewConfig(&awssume.NewConfigOpts{
				Fs:   afero.NewOsFs(),
				Path: path.Join(curUser.HomeDir, awssume.DefaultConfigFilePath),
			})
			if err != nil {
				return fmt.Errorf(awssume.ErrNewConfig, err)
			}

			tw.Write([]byte(strings.Join([]string{
				"ALIAS", "ARN", "SESSION_NAME",
			}, "\t") + "\n"))

			for _, r := range cfg.GetRoles() {
				tw.Write([]byte(strings.Join([]string{
					r.GetAlias(), r.GetARN().String(), r.GetSessionName(),
				}, "\t") + "\n"))
			}

			return tw.Flush()
		},
	}

	convertCmd := &cobra.Command{
		Use:     "convert [format]",
		Aliases: []string{"c", "conv"},
		Short:   "Convert configuration between formats",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errTooFewArguments
			}

			fmtExt := args[0]

			curUser, err := user.Current()
			if err != nil {
				return fmt.Errorf(errCurrentUser, err)
			}

			cfg, err := awssume.NewConfig(&awssume.NewConfigOpts{
				Fs:   afero.NewOsFs(),
				Path: path.Join(curUser.HomeDir, awssume.DefaultConfigFilePath),
			})
			if err != nil {
				return fmt.Errorf(awssume.ErrNewConfig, err)
			}

			toRemove := strings.Join([]string{
				cfg.GetPath(), cfg.GetFormat().String(),
			}, ".")

			var cfgFmt awssume.ConfigFormat
			cfgFmt.FromExt(fmtExt)
			cfg.SetFormat(cfgFmt)

			if err := cfg.Save(); err != nil {
				return fmt.Errorf(
					awssume.ErrWritingToFile,
					strings.Join([]string{
						cfg.GetPath(), cfg.GetFormat().String(),
					}, "."),
					err,
				)
			}

			return afero.NewOsFs().Remove(toRemove)
		},
	}

	addCmd := &cobra.Command{
		Use:     "add [role] [alias] [session_name]",
		Short:   "Add a new Role",
		Aliases: []string{"a"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 3 {
				return errTooFewArguments
			}

			curUser, err := user.Current()
			if err != nil {
				return fmt.Errorf(errCurrentUser, err)
			}

			cfg, err := awssume.NewConfig(&awssume.NewConfigOpts{
				Fs:   afero.NewOsFs(),
				Path: path.Join(curUser.HomeDir, awssume.DefaultConfigFilePath),
			})
			if err != nil {
				return fmt.Errorf(awssume.ErrNewConfig, err)
			}

			roleARN, err := awssume.ParseARN(args[0])
			if err != nil {
				return err
			}

			if err := cfg.AddRole(&awssume.Role{
				ARN: &roleARN, Alias: args[1], SessionName: args[2],
			}); err != nil {
				return err
			}

			return cfg.Save()
		},
	}

	var sessionDuration int64
	execCmd := &cobra.Command{
		Use:     "exec",
		Aliases: []string{"e", "ex", "exe"},
		Short:   "Execute a subprocess with Role credentials as environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errTooFewArguments
			}

			alias := args[0]

			dashIdx := cmd.ArgsLenAtDash()
			if dashIdx == -1 {
				return awssume.ErrCommandMissing
			}

			command := args[1]
			arguments := []string{}
			if len(args) > 1 {
				arguments = args[2:]
			}

			curUser, err := user.Current()
			if err != nil {
				return fmt.Errorf(errCurrentUser, err)
			}

			cfg, err := awssume.NewConfig(&awssume.NewConfigOpts{
				Fs:   afero.NewOsFs(),
				Path: path.Join(curUser.HomeDir, awssume.DefaultConfigFilePath),
			})
			if err != nil {
				return fmt.Errorf(awssume.ErrNewConfig, err)
			}

			return cfg.ExecRole(alias, sessionDuration, command, arguments)
		},
	}

	execCmd.PersistentFlags().Int64VarP(
		&sessionDuration,
		"session-duration",
		"d",
		60*60,
		"The duration of the STS Session when the Role is assumed",
	)

	rootCmd.AddCommand(versionCmd, listCmd, convertCmd, addCmd, execCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
