package completion

import (
	"context"

	"github.com/metal-stack/api/go/client"
	"github.com/spf13/cobra"
)

type Completion struct {
	Client  client.Client
	Project string
	Ctx     context.Context
}

func OutputFormatListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "wide", "markdown", "json", "yaml", "template"}, cobra.ShellCompDirectiveNoFileComp
}
