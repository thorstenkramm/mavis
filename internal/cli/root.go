// Package cli provides the Mavis command-line interface.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/server"
)

// ServeFunc starts and runs a serving session.
type ServeFunc func(ctx context.Context, root string, configuration config.Config, stdout, stderr io.Writer) error

// NewRootCommand constructs the Mavis root command.
func NewRootCommand() *cobra.Command {
	return NewRootCommandWithServe(server.Serve)
}

// NewRootCommandWithServe constructs a root command with an injectable server.
func NewRootCommandWithServe(serve ServeFunc) *cobra.Command {
	root := &cobra.Command{
		Use:           "mavis",
		Short:         "Serve Markdown documentation locally",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.AddCommand(newServeCommand(serve))
	return root
}

// Main runs Mavis with process-style arguments and returns an exit code.
func Main(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	command := NewRootCommand()
	command.SetArgs(args)
	command.SetOut(stdout)
	command.SetErr(stderr)
	if err := command.ExecuteContext(ctx); err != nil {
		if _, writeErr := fmt.Fprintln(stderr, "Error:", err); writeErr != nil {
			return 1
		}
		return 1
	}
	return 0
}

func newServeCommand(serve ServeFunc) *cobra.Command {
	var host string
	var port int
	var socket string
	var exclude []string
	var includeHidden bool

	command := &cobra.Command{
		Use:   "serve <directory>",
		Short: "Serve a directory of Markdown documentation",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			root, err := validateRoot(args[0])
			if err != nil {
				return err
			}
			fileConfiguration, err := config.Load(root)
			if err != nil {
				return err
			}
			overrides := config.Overrides{}
			if command.Flags().Changed("host") {
				overrides.Host = &host
			}
			if command.Flags().Changed("port") {
				overrides.Port = &port
			}
			if command.Flags().Changed("socket") {
				overrides.Socket = &socket
			}
			if command.Flags().Changed("include-hidden") {
				overrides.IncludeHidden = &includeHidden
			}
			if command.Flags().Changed("exclude") {
				overrides.Exclude = append([]string(nil), exclude...)
			}
			configuration, err := config.Merge(fileConfiguration, overrides)
			if err != nil {
				return err
			}
			return serve(command.Context(), root, configuration, command.OutOrStdout(), command.ErrOrStderr())
		},
	}
	flags := command.Flags()
	flags.StringVar(&host, "host", config.DefaultHost, "TCP hostname or address")
	flags.IntVar(&port, "port", config.DefaultPort, "TCP port")
	flags.StringVar(&socket, "socket", "", "Unix-socket path instead of TCP")
	flags.StringArrayVar(&exclude, "exclude", nil, "Root-relative wildcard exclusion (repeatable)")
	flags.BoolVar(&includeHidden, "include-hidden", false, "Include dotfiles and dot directories unless excluded")
	return command
}

func validateRoot(directory string) (string, error) {
	root, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("resolve content root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("invalid content root %q: %w", directory, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("invalid content root %q: not a directory", directory)
	}
	handle, err := os.Open(root)
	if err != nil {
		return "", fmt.Errorf("invalid content root %q: directory is not readable: %w", directory, err)
	}
	if err := handle.Close(); err != nil {
		return "", fmt.Errorf("close content root %q: %w", directory, err)
	}
	return root, nil
}
