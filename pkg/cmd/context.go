package cmd

import (
	"github.com/spf13/cobra"
)

type CmdConfig struct {
	Use               string
	ValidArgsFunction cobra.CompletionFunc
	Example           string
	RunE              func(cmd *cobra.Command, args []string) error
	Short, Long       string
	Aliases           []string
	MutateFn          func(cmd *cobra.Command)
}

func ContextBaseCmd(c *CmdConfig) *cobra.Command {
	if c.Use == "" {
		c.Use = "context <name>"
	}
	if c.Short == "" {
		c.Short = "manage cli contexts"
	}
	if c.Long == "" {
		c.Long = "context defines the backend to talk to. You can switch back and forth with \"-\" as a shortcut to the last used context."
	}
	if c.Aliases == nil {
		c.Aliases = []string{"ctx"}
	}
	if c.Example == "" {
		c.Example = "Please refer to the documentation for examples on how to use contexts."
	}

	return toCommand(c)
}

func ContextAddCmd(c *CmdConfig) *cobra.Command {
	if c.Use == "" {
		c.Use = "add <context-name>"
	}
	if c.Short == "" {
		c.Short = "add a cli context"
	}
	if c.Aliases == nil {
		c.Aliases = []string{"create"}
	}

	return toCommand(c)
}

func ContextRemoveCmd(c *CmdConfig) *cobra.Command {
	if c.Use == "" {
		c.Use = "remove <context-name>"
	}
	if c.Short == "" {
		c.Short = "remove a cli context"
	}
	if c.Aliases == nil {
		c.Aliases = []string{"delete", "rm"}
	}

	return toCommand(c)
}

func ContextUpdateCmd(c *CmdConfig) *cobra.Command {
	if c.Use == "" {
		c.Use = "update <context-name>"
	}
	if c.Short == "" {
		c.Short = "update a cli context"
	}

	return toCommand(c)
}

func ContextListCmd(c *CmdConfig) *cobra.Command {
	if c.Short == "" {
		c.Short = "list the configured cli contexts"
	}
	if c.Use == "" {
		c.Use = "list"
	}

	return toCommand(c)
}

func toCommand(c *CmdConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:               c.Use,
		Aliases:           c.Aliases,
		Short:             c.Short,
		Long:              c.Long,
		Example:           c.Example,
		ValidArgsFunction: c.ValidArgsFunction,
		RunE:              c.RunE,
	}

	if c.MutateFn != nil {
		c.MutateFn(cmd)
	}

	return cmd
}
