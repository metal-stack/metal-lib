package cmd

import "github.com/spf13/cobra"

type ContextConfig struct {
	ValidArgsFunction cobra.CompletionFunc
	Example           string
	RunE              func(cmd *cobra.Command, args []string) error
	Short, Long       string
}

func NewContextCmd(c *ContextConfig) *cobra.Command {
	if c.Short == "" {
		c.Short = "Manage contexts"
	}
	if c.Long == "" {
		c.Long = "context defines the backend to talks to. You can switch back and forth with \"-\" as a shortcut to the last used context."
	}
	if c.Example == "" {
		c.Example = "Please refer to the documentation for examples on how to use contexts."
	}

	contextCmd := &cobra.Command{
		Use:               "context <name>",
		Aliases:           []string{"ctx"},
		Short:             c.Short,
		Long:              c.Long,
		ValidArgsFunction: c.ValidArgsFunction,
		Example:           c.Example,
		RunE:              c.RunE,
	}

	return contextCmd
}
