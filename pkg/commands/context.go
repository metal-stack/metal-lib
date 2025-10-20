package commands

import (
	"fmt"

	"github.com/fatih/color"

	"github.com/metal-stack/metal-lib/pkg/commands/helpers/sorters"
	"github.com/metal-stack/metal-lib/pkg/commands/types"
	"github.com/metal-stack/metal-lib/pkg/genericcli"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ctx struct {
	c *types.Config
}

// func test() {
// 	cmdsConfig := genericcli.CmdsConfig[*cobra.Command, int, *types.Context]{ // ???
// 		OnlyCmds: map[genericcli.DefaultCmd]bool{
// 			genericcli.ListCmd:     true,
// 			genericcli.DescribeCmd: false,
// 			genericcli.CreateCmd:   true,
// 			genericcli.UpdateCmd:   true,
// 			genericcli.DeleteCmd:   true,
// 			genericcli.ApplyCmd:    false,
// 			genericcli.EditCmd:     false,
// 		},
// 		CreateRequestFromCLI: c.add,
// 		UpdateRequestFromCLI: c.update,
// 		DeleteRequestFromCLI: c.remove,
// 		ListRequestFromCLI:   c.list,
// 		Sorter:               sorters.ContextSorter(),
// 		ListPrinter:          printers.ListPrinter[*types.Context],
// 		Singular:             "context",
// 		Plural:               "contexts",
// 		Aliases:              []string{"ctx"},
// 		Description:          "manage cli contexts. You can switch contexts back and forth with \"-\"",
// 	}

// 	genericcli.NewCmds(cmdsConfig)
// }

func NewContextCmd(c *types.Config) *cobra.Command {
	w := &ctx{
		c: c,
	}

	contextCmd := &cobra.Command{
		Use:     "context",
		Aliases: []string{"ctx"},
		Short:   "manage cli contexts",
		Long:    "you can switch back and forth contexts with \"-\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return w.list()
			}

			return w.set(args)
		},
	}

	contextListCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "list the configured cli contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return w.list()
		},
	}
	contextSwitchCmd := &cobra.Command{
		Use:     "switch <context-name>",
		Short:   "switch the cli context",
		Long:    "you can switch back and forth contexts with \"-\"",
		Aliases: []string{"set", "sw"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return w.set(args)
		},
		ValidArgsFunction: c.ContextListCompletion,
	}
	contextShortCmd := &cobra.Command{
		Use:   "show-current",
		Short: "prints the current context name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return w.short()
		},
	}
	contextSetProjectCmd := &cobra.Command{
		Use:   "set-project <project-id>",
		Short: "sets the default project to act on for cli commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return w.setProject(args)
		},
		ValidArgsFunction: c.Completion.ProjectListCompletion,
	}
	contextRemoveCmd := &cobra.Command{
		Use:     "remove <context-name>",
		Aliases: []string{"rm", "delete"},
		Short:   "remove a cli context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return w.remove(args)
		},
		ValidArgsFunction: c.ContextListCompletion,
	}

	contextAddCmd := &cobra.Command{
		Use:     "add <context-name>",
		Aliases: []string{"create"},
		Short:   "add a cli context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return w.add(args)
		},
	}
	contextAddCmd.Flags().String("api-url", "", "sets the api-url for this context")
	contextAddCmd.Flags().String("api-token", "", "sets the api-token for this context")
	contextAddCmd.Flags().String("default-project", "", "sets a default project to act on")
	contextAddCmd.Flags().Duration("timeout", 0, "sets a default request timeout")
	contextAddCmd.Flags().Bool("activate", false, "immediately switches to the new context")
	contextAddCmd.Flags().String("provider", "", "sets the login provider for this context")

	genericcli.Must(contextAddCmd.MarkFlagRequired("api-token"))

	contextUpdateCmd := &cobra.Command{
		Use:   "update <context-name>",
		Short: "update a cli context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return w.update(args)
		},
		ValidArgsFunction: c.ContextListCompletion,
	}
	contextUpdateCmd.Flags().String("api-url", "", "sets the api-url for this context")
	contextUpdateCmd.Flags().String("api-token", "", "sets the api-token for this context")
	contextUpdateCmd.Flags().String("default-project", "", "sets a default project to act on")
	contextUpdateCmd.Flags().Duration("timeout", 0, "sets a default request timeout")
	contextUpdateCmd.Flags().Bool("activate", false, "immediately switches to the new context")
	contextUpdateCmd.Flags().String("provider", "", "sets the login provider for this context")

	genericcli.Must(contextUpdateCmd.RegisterFlagCompletionFunc("default-project", c.Completion.ProjectListCompletion))

	contextCmd.AddCommand(
		contextListCmd,
		contextAddCmd,
		contextRemoveCmd,
		contextUpdateCmd,
		contextSwitchCmd,
		contextShortCmd,
		contextSetProjectCmd,
	)

	return contextCmd
}

func (c *ctx) list() error {
	ctxs, err := c.c.GetContexts()
	if err != nil {
		return err
	}

	err = sorters.ContextSorter().SortBy(ctxs.Contexts)
	if err != nil {
		return err
	}

	return c.c.ListPrinter.Print(ctxs)
}

func (c *ctx) short() error {
	ctxs, err := c.c.GetContexts()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprint(c.c.Out, ctxs.CurrentContext)

	return nil
}

func (c *ctx) add(args []string) error {
	name, err := genericcli.GetExactlyOneArg(args)
	if err != nil {
		return fmt.Errorf("no context name given")
	}

	ctxs, err := c.c.GetContexts()
	if err != nil {
		return err
	}

	_, ok := ctxs.Get(name)
	if ok {
		return fmt.Errorf("context with name %q already exists", name)
	}

	ctx := &types.Context{
		Name:           name,
		ApiURL:         pointer.PointerOrNil(viper.GetString("api-url")),
		Token:          viper.GetString("api-token"),
		DefaultProject: viper.GetString("default-project"),
		Timeout:        pointer.PointerOrNil(viper.GetDuration("timeout")),
		Provider:       viper.GetString("provider"),
	}

	ctxs.Contexts = append(ctxs.Contexts, ctx)

	if viper.GetBool("activate") || ctxs.CurrentContext == "" {
		ctxs.PreviousContext = ctxs.CurrentContext
		ctxs.CurrentContext = ctx.Name
	}

	err = c.c.WriteContexts(ctxs)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.c.Out, "%s added context \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.Name))

	return nil
}

func (c *ctx) update(args []string) error {
	name, err := genericcli.GetExactlyOneArg(args)
	if err != nil {
		return fmt.Errorf("no context name given")
	}

	ctxs, err := c.c.GetContexts()
	if err != nil {
		return err
	}

	ctx, ok := ctxs.Get(name)
	if !ok {
		return fmt.Errorf("no context with name %q found", name)
	}

	if viper.IsSet("api-url") {
		ctx.ApiURL = pointer.PointerOrNil(viper.GetString("api-url"))
	}
	if viper.IsSet("api-token") {
		ctx.Token = viper.GetString("api-token")
	}
	if viper.IsSet("default-project") {
		ctx.DefaultProject = viper.GetString("default-project")
	}
	if viper.IsSet("timeout") {
		ctx.Timeout = pointer.PointerOrNil(viper.GetDuration("timeout"))
	}
	if viper.IsSet("provider") {
		ctx.Provider = viper.GetString("provider")
	}
	if viper.GetBool("activate") {
		ctxs.PreviousContext = ctxs.CurrentContext
		ctxs.CurrentContext = ctx.Name
	}

	err = c.c.WriteContexts(ctxs)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.c.Out, "%s updated context \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.Name))

	return nil
}

func (c *ctx) remove(args []string) error {
	name, err := genericcli.GetExactlyOneArg(args)
	if err != nil {
		return fmt.Errorf("no context name given")
	}

	ctxs, err := c.c.GetContexts()
	if err != nil {
		return err
	}

	ctx, ok := ctxs.Get(name)
	if !ok {
		return fmt.Errorf("no context with name %q found", name)
	}

	ctxs.Delete(ctx.Name)

	err = c.c.WriteContexts(ctxs)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.c.Out, "%s removed context \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.Name))

	return nil
}

func (c *ctx) set(args []string) error {
	wantCtx, err := genericcli.GetExactlyOneArg(args)
	if err != nil {
		return fmt.Errorf("no context name given")
	}

	ctxs, err := c.c.GetContexts()
	if err != nil {
		return err
	}

	if wantCtx == "-" {
		prev := ctxs.PreviousContext
		if prev == "" {
			return fmt.Errorf("no previous context found")
		}

		curr := ctxs.CurrentContext
		ctxs.PreviousContext = curr
		ctxs.CurrentContext = prev
	} else {
		nextCtx := wantCtx
		_, ok := ctxs.Get(nextCtx)
		if !ok {
			return fmt.Errorf("context %s not found", nextCtx)
		}
		if nextCtx == ctxs.CurrentContext {
			_, _ = fmt.Fprintf(c.c.Out, "%s context \"%s\" already active\n", color.GreenString("✔"), color.GreenString(ctxs.CurrentContext))
			return nil
		}
		ctxs.PreviousContext = ctxs.CurrentContext
		ctxs.CurrentContext = nextCtx
	}

	err = c.c.WriteContexts(ctxs)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.c.Out, "%s switched context to \"%s\"\n", color.GreenString("✔"), color.GreenString(ctxs.CurrentContext))

	return nil
}

func (c *ctx) setProject(args []string) error {
	project, err := genericcli.GetExactlyOneArg(args)
	if err != nil {
		return err
	}

	ctxs, err := c.c.GetContexts()
	if err != nil {
		return err
	}

	ctx, ok := ctxs.Get(c.c.Context.Name)
	if !ok {
		return fmt.Errorf("no context currently active")
	}

	ctx.DefaultProject = project

	err = c.c.WriteContexts(ctxs)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.c.Out, "%s switched context default project to \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.DefaultProject))

	return nil
}
