package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/metal-stack/metal-lib/pkg/genericcli"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Contexts contains all configuration contexts
type contexts struct {
	CurrentContext  string     `json:"current-context"`
	PreviousContext string     `json:"previous-context"`
	Contexts        []*Context `json:"contexts"`
}

// Context configure
type Context struct {
	Name           string         `json:"name"`
	ApiURL         *string        `json:"api-url,omitempty"`
	Token          string         `json:"api-token"`
	DefaultProject string         `json:"default-project"`
	Timeout        *time.Duration `json:"timeout,omitempty"`
	Provider       string         `json:"provider"`
}

func (c *Context) GetCurrentContext() string {
	return contexts{}.CurrentContext
}

// // ContextClient defines the interface for context operations
// type ContextClient interface {
// 	List() ([]*Context, error)
// 	Get(name string) (*Context, error)
// 	Create(rq **Context) (*Context, error)
// 	Update(rq *ContextUpdateRequest) (*Context, error)
// 	Delete(name string) (*Context, error)
// 	Switch(name string) (*Context, error)

// 	Convert(r string) (string, **Context, *ContextUpdateRequest, error)
// }

// // contextClientImpl implements the ContextClient interface
// // This would typically interact with your config file or backend
// type contextClientImpl struct {
// 	configPath string
// 	// Add other fields for managing context state
// }

// // NewContextClient creates a new context client
// func NewContextClient(configPath string) ContextClient {
// 	return &contextClientImpl{
// 		configPath: configPath,
// 	}
// }

// func (c *contextClientImpl) Convert(r string) (string, **Context, *ContextUpdateRequest, error) {
// 	// TODO: Implement loading contexts from config file
// 	// This is a placeholder implementation
// 	return "", nil, nil, fmt.Errorf("Not implemented")
// }

// func (c *contextClientImpl) List() ([]*Context, error) {
// 	// TODO: Implement loading contexts from config file
// 	// This is a placeholder implementation
// 	return []*Context{
// 		{Name: "default", Backend: "http://localhost:8080", IsActive: true},
// 		{Name: "production", Backend: "https://api.example.com", IsActive: false},
// 	}, nil
// }

// func (c *contextClientImpl) Get(name string) (*Context, error) {
// 	// TODO: Implement getting a specific context
// 	contexts, err := c.List()
// 	if err != nil {
// 		return nil, err
// 	}
// 	for _, ctx := range contexts {
// 		if ctx.Name == name {
// 			return ctx, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("context %q not found", name)
// }

// func (c *contextClientImpl) Create(rq **Context) (*Context, error) {
// 	// TODO: Implement context creation
// 	return &Context{
// 		Name:     rq.Name,
// 		Backend:  rq.Backend,
// 		IsActive: false,
// 	}, nil
// }

// func (c *contextClientImpl) Update(rq *ContextUpdateRequest) (*Context, error) {
// 	// TODO: Implement context update
// 	ctx, err := c.Get(rq.Name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if rq.Backend != "" {
// 		ctx.Backend = rq.Backend
// 	}
// 	return ctx, nil
// }

// func (c *contextClientImpl) Delete(name string) (*Context, error) {
// 	// TODO: Implement context deletion
// 	ctx, err := c.Get(name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return ctx, nil
// }

// func (c *contextClientImpl) Switch(name string) (*Context, error) {
// 	// TODO: Implement context switching
// 	// Handle the special "-" case for switching to last used context
// 	if name == "-" {
// 		// Load and return last used context
// 	}
// 	ctx, err := c.Get(name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	ctx.IsActive = true
// 	return ctx, nil
// }

// contextTablePrinter creates a table printer for contexts
// func contextTablePrinter() printers.Printer {
// 	return printers.NewTablePrinter(&printers.TablePrinterConfig{
// 		ToHeaderRow: func(data []*Context) []string {
// 			return []string{"Name", "Backend", "Active"}
// 		},
// 		ToRow: func(data *Context) []string {
// 			active := ""
// 			if data.IsActive {
// 				active = "*"
// 			}
// 			return []string{data.Name, data.Backend, active}
// 		},
// 	})
// }

// type Subcommand genericcli.DefaultCmd

// const (
// 	SUBCMD_LIST       Subcommand = Subcommand(genericcli.ListCmd)
// 	SUBCMD_ADD        Subcommand = Subcommand(genericcli.CreateCmd)
// 	SUBCMD_DELETE     Subcommand = Subcommand(genericcli.DeleteCmd)
// 	SUBCMD_UPDATE     Subcommand = Subcommand(genericcli.UpdateCmd)
// 	SUBCMD_SHORT      Subcommand = "ctx"
// 	SUBCMD_SETPROJECT Subcommand = "set-project"
// 	SUBCMD_SWITCH     Subcommand = "switch"
// 	// DescribeCmd DefaultCmd = "describe"
// 	// ApplyCmd    DefaultCmd = "apply"
// 	// EditCmd     DefaultCmd = "edit"
// )

// type ContextCmdConfig struct {
// 	Use               string
// 	ValidArgsFunction cobra.CompletionFunc
// 	Example           string
// 	RunE              func(cmd *cobra.Command, args []string) error
// 	Short, Long       string
// 	Aliases           []string
// 	MutateFn          func(cmd *cobra.Command)
// }

type ContextConfig struct {
	ConfigPath string
	BinaryName string
	Fs         afero.Fs

	// I/O
	DescribePrinter       func() printers.Printer
	ListPrinter           func() printers.Printer
	In                    io.Reader
	Out                   io.Writer
	ProjectListCompletion cobra.CompletionFunc
}

// t function mimics the ternary operator
// func t[T any](condition bool, trueVal, falseVal T) T {
// 	if condition {
// 		return trueVal
// 	}
// 	return falseVal
// }

// func (c *ContextConfig[Context]) setDefaultIfMissing() {
// 	c.Singular = cmp.Or(c.Singular, "context")
// 	c.Plural = cmp.Or(c.Plural, "contexts")
// 	// c.DescribePrinter = cmp.Or(c.DescribePrinter, func () printers.Printer { return printers.})
// 	// c.DescribePrinter = cmp.Or(c.DescribePrinter, func () printers.Printer { return printers.})
// 	c.In = cmp.Or(c.In, io.Reader(os.Stdin))
// 	c.Out = cmp.Or(c.Out, io.Writer(os.Stdout))
// }

type cliWrapper struct {
	cfg *ContextConfig
}

// NewContextCmd creates the context command tree using genericcli
func NewContextCmd(c *ContextConfig) *cobra.Command {
	// Create the generic CLI wrapper

	cmd := genericcli.NewCmds(&genericcli.CmdsConfig[
		*Context,
		*Context,
		*Context,
	]{
		GenericCLI: genericcli.NewGenericCLI(&cliWrapper{
			cfg: c,
		}),
		BinaryName:      c.BinaryName,
		Singular:        "context",
		Plural:          "contexts",
		Description:     "context defines the backend to talk to. You can switch back and forth with \"-\" as a shortcut to the last used context.",
		Aliases:         []string{"ctx"},
		Args:            []string{"name"},
		DescribePrinter: c.DescribePrinter,
		ListPrinter:     c.ListPrinter,

		// ListCmdMutateFn:   nil,
		CreateCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Flags().String("api-url", "", "sets the api-url for this context")
			cmd.Flags().String("api-token", "", "sets the api-token for this context")
			cmd.Flags().String("default-project", "", "sets a default project to act on")
			cmd.Flags().Duration("timeout", 0, "sets a default request timeout")
			cmd.Flags().Bool("activate", false, "immediately switches to the new context")
			cmd.Flags().String("provider", "", "sets the login provider for this context")

			genericcli.Must(cmd.MarkFlagRequired("api-token"))
		},
		UpdateCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Flags().String("api-url", "", "sets the api-url for this context")
			cmd.Flags().String("api-token", "", "sets the api-token for this context")
			cmd.Flags().String("default-project", "", "sets a default project to act on")
			cmd.Flags().Duration("timeout", 0, "sets a default request timeout")
			cmd.Flags().Bool("activate", false, "immediately switches to the new context")
			cmd.Flags().String("provider", "", "sets the login provider for this context")

			genericcli.Must(cmd.RegisterFlagCompletionFunc("default-project", c.ProjectListCompletion))
		},

		// Custom create function from CLI flags
		CreateRequestFromCLI: func() (*Context, error) {
			return &Context{}, nil
		},

		// Custom update function from CLI flags
		UpdateRequestFromCLI: func(args []string) (*Context, error) {
			return &Context{}, nil
		},

		// Add completion for context names
		// ValidArgsFn: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// 	if len(args) > 0 {
		// 		return nil, cobra.ShellCompDirectiveNoFileComp
		// 	}
		// 	contexts, err := client.List()
		// 	if err != nil {
		// 		return nil, cobra.ShellCompDirectiveError
		// 	}
		// 	names := make([]string, len(contexts))
		// 	for i, ctx := range contexts {
		// 		names[i] = ctx.Name
		// 	}
		// 	return names, cobra.ShellCompDirectiveNoFileComp
		// },

		In:  c.In,
		Out: c.Out,
	})

	// Add custom "switch" command (not part of genericcli defaults)
	// switchCmd := &cobra.Command{
	// 	Use:               "switch <context-name>",
	// 	Short:             "switch the cli context",
	// 	Long:              "switch the cli context. Use \"-\" to switch to the previously used context.",
	// 	Aliases:           []string{"set", "sw"},
	// 	Args:              cobra.ExactArgs(1),
	// 	RunE:              c.SwitchCmdRunE,
	// 	ValidArgsFunction: c.ContextListCompletion,
	// }

	// setProjectCmd := &cobra.Command{
	// 	Use:               "set-project <project-id>",
	// 	Short:             "sets the default project to act on for cli commands",
	// 	RunE:              c.SetProjectCmdRunE,
	// 	ValidArgsFunction: c.ProjectListCompletion,
	// }

	// cmd.AddCommand(
	// 	switchCmd,
	// 	setProjectCmd,
	// )

	return cmd
}

func (c *cliWrapper) Get(id string) (*Context, error) {
	return nil, fmt.Errorf("testGet")
}

func (c *cliWrapper) List() ([]*Context, error) {
	return nil, fmt.Errorf("you need to create a context first")
}

func (c *cliWrapper) Create(rq *Context) (*Context, error) {
	return nil, fmt.Errorf("testCreate")
}

func (c *cliWrapper) Update(rq *Context) (*Context, error) {
	return nil, fmt.Errorf("testUpdate")
}

func (c *cliWrapper) Delete(id string) (*Context, error) {
	return nil, fmt.Errorf("testDelete")
}

func (c *cliWrapper) Convert(r *Context) (string, *Context, *Context, error) {
	return "Yay!", &Context{}, &Context{}, fmt.Errorf("testGet")
}

// // contextCLIWrapper wraps the ContextClient to implement the genericcli interfaces
// type contextCLIWrapper struct {
// 	client ContextClient
// }

// // Get implements the getter interface for genericcli
// func (w *contextCLIWrapper) Get(id string) (*Context, error) {
// 	return w.client.Get(id)
// }

// // List implements the lister interface for genericcli
// func (w *contextCLIWrapper) List() ([]*Context, error) {
// 	return w.client.List()
// }

// // Create implements the creator interface for genericcli
// func (w *contextCLIWrapper) Create(rq *Context) (*Context, error) {
// 	return w.client.Create(&rq)
// }

// // Update implements the updater interface for genericcli
// func (w *contextCLIWrapper) Update(rq ContextUpdateRequest) (*Context, error) {
// 	return w.client.Update(&rq)
// }

// // Delete implements the deleter interface for genericcli
// func (w *contextCLIWrapper) Delete(id string) (*Context, error) {
// 	return w.client.Delete(id)
// }

// // Convert converts a Context to itself (identity conversion)
// func (w *contextCLIWrapper) Convert(r *Context) *Context {
// 	return r
// }

// Example usage in your main.go or root command:
/*
func init() {
	client := NewContextClient(viper.GetString("config"))
	contextCmd := NewContextCmd(client, "yourapp")
	rootCmd.AddCommand(contextCmd)
}

// This provides these commands:
// yourapp context list                    - list all contexts
// yourapp context describe <name>         - show details of a context
// yourapp context create                  - create a new context (with --backend flag)
// yourapp context update <name>           - update a context (with --backend flag)
// yourapp context delete <name>           - delete a context
// yourapp context apply -f contexts.yaml  - bulk create/update from file
// yourapp context edit <name>             - edit context in editor
// yourapp context switch <name>           - switch to a context (custom command)
*/

// type CmdConfig struct {
// 	Use               string
// 	ValidArgsFunction cobra.CompletionFunc
// 	Example           string
// 	RunE              func(cmd *cobra.Command, args []string) error
// 	Short, Long       string
// 	Aliases           []string
// 	MutateFn          func(cmd *cobra.Command)
// }

// func ContextBaseCmd(c *CmdConfig) *cobra.Command {
// 	if c.Use == "" {
// 		c.Use = "context <name>"
// 	}
// 	if c.Short == "" {
// 		c.Short = "manage cli contexts"
// 	}
// 	if c.Long == "" {
// 		c.Long = "context defines the backend to talk to. You can switch back and forth with \"-\" as a shortcut to the last used context."
// 	}
// 	if c.Aliases == nil {
// 		c.Aliases = []string{"ctx"}
// 	}
// 	if c.Example == "" {
// 		c.Example = "Please refer to the documentation for examples on how to use contexts."
// 	}

// 	return toCommand(c)
// }

// func ContextAddCmd(c *CmdConfig) *cobra.Command {
// 	if c.Use == "" {
// 		c.Use = "add <context-name>"
// 	}
// 	if c.Short == "" {
// 		c.Short = "add a cli context"
// 	}
// 	if c.Aliases == nil {
// 		c.Aliases = []string{"create"}
// 	}

// 	return toCommand(c)
// }

// func ContextRemoveCmd(c *CmdConfig) *cobra.Command {
// 	if c.Use == "" {
// 		c.Use = "remove <context-name>"
// 	}
// 	if c.Short == "" {
// 		c.Short = "remove a cli context"
// 	}
// 	if c.Aliases == nil {
// 		c.Aliases = []string{"delete", "rm"}
// 	}

// 	return toCommand(c)
// }

// func ContextUpdateCmd(c *CmdConfig) *cobra.Command {
// 	if c.Use == "" {
// 		c.Use = "update <context-name>"
// 	}
// 	if c.Short == "" {
// 		c.Short = "update a cli context"
// 	}

// 	return toCommand(c)
// }

// func ContextListCmd(c *CmdConfig) *cobra.Command {
// 	if c.Short == "" {
// 		c.Short = "list the configured cli contexts"
// 	}
// 	if c.Use == "" {
// 		c.Use = "list"
// 	}

// 	return toCommand(c)
// }

// func toCommand(c *CmdConfig) *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:               c.Use,
// 		Aliases:           c.Aliases,
// 		Short:             c.Short,
// 		Long:              c.Long,
// 		Example:           c.Example,
// 		ValidArgsFunction: c.ValidArgsFunction,
// 		RunE:              c.RunE,
// 	}

// 	if c.MutateFn != nil {
// 		c.MutateFn(cmd)
// 	}

// 	return cmd
// }

type tablePrinter struct{}

func (t *tablePrinter) contextTable(data []*Context, wide bool) ([]string, [][]string, error) {
	var (
		header = []string{"", "Name", "Provider", "Default Project"}
		rows   [][]string
	)

	if wide {
		header = append(header, "API URL")
	}

	for _, c := range data {
		active := ""
		if c.Name == c.GetCurrentContext() {
			active = color.GreenString("✔")
		}

		row := []string{active, c.Name, c.Provider, c.DefaultProject}
		if wide {
			url := pointer.SafeDeref(c.ApiURL)
			if url == "" {
				url = viper.GetString("api-url")
			}

			row = append(row, url)
		}

		rows = append(rows, row)
	}

	return header, rows, nil
}
