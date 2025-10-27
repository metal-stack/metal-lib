package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/fatih/color"
	"github.com/metal-stack/metal-lib/pkg/genericcli"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	keyName           = "name"
	keyApiUrl         = "api-url"
	keyApiToken       = "api-token"
	keyDefaultProject = "default-project"
	keyTimeout        = "timeout"
	keyActivate       = "activate"
	keyProvider       = "provider"
	keyConfig         = "config"

	defaultConfigName = "config.yaml"
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

type ContextConfig struct {
	BinaryName    string
	ConfigName    string
	ConfigDirName string
	Fs            afero.Fs

	// I/O
	DescribePrinter       func() printers.Printer
	ListPrinter           func() printers.Printer
	In                    io.Reader
	Out                   io.Writer
	ProjectListCompletion cobra.CompletionFunc
}

type cliWrapper struct {
	cfg *ContextConfig
}

// NewContextCmd creates the context command tree using genericcli
func NewContextCmd(c *ContextConfig) *cobra.Command {
	// TODO check nils
	c.ConfigName = cmp.Or(c.ConfigName, string(defaultConfigName))
	c.Out = cmp.Or(c.Out, io.Writer(os.Stdout))
	c.In = cmp.Or(c.In, io.Reader(os.Stdin))

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
		Args:            []string{"name"}, // TODO is this needed when using a flag? (--name)
		DescribePrinter: c.DescribePrinter,
		ListPrinter:     func() printers.Printer { return &tablePrinter{} },

		// ListCmdMutateFn:   nil,
		CreateCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Flags().String(keyName, "", "sets the name of the context")
			cmd.Flags().String(keyApiUrl, "", "sets the api-url for this context")
			cmd.Flags().String(keyApiToken, "", "sets the api-token for this context")
			cmd.Flags().String(keyDefaultProject, "", "sets a default project to act on")
			cmd.Flags().Duration(keyTimeout, 0, "sets a default request timeout")
			cmd.Flags().Bool(keyActivate, false, "immediately switches to the new context")
			cmd.Flags().String(keyProvider, "", "sets the login provider for this context")

			genericcli.Must(cmd.MarkFlagRequired(keyName))
			genericcli.Must(cmd.MarkFlagRequired(keyApiToken))
		},
		UpdateCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Flags().String(keyApiUrl, "", "sets the api-url for this context")
			cmd.Flags().String(keyApiToken, "", "sets the api-token for this context")
			cmd.Flags().String(keyDefaultProject, "", "sets a default project to act on")
			cmd.Flags().Duration(keyTimeout, 0, "sets a default request timeout")
			cmd.Flags().Bool(keyActivate, false, "immediately switches to the new context")
			cmd.Flags().String(keyProvider, "", "sets the login provider for this context")

			genericcli.Must(cmd.RegisterFlagCompletionFunc(keyDefaultProject, c.ProjectListCompletion))
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
	ctxs, err := c.cfg.GetContexts()
	if err != nil {
		return nil, err
	}

	// err = ContextSorter().SortBy(ctxs.Contexts)
	// if err != nil {
	// 	return err
	// }

	return ctxs.Contexts, nil
	// return nil, fmt.Errorf("you need to create a context first")
}

func (c *cliWrapper) Create(rq *Context) (*Context, error) {
	name := viper.GetString(keyName)
	ctxs, err := c.cfg.GetContexts()
	if err != nil {
		return nil, err
	}

	ctx := &Context{
		Name:           name,
		ApiURL:         pointer.PointerOrNil(viper.GetString(keyApiUrl)),
		Token:          viper.GetString(keyApiToken),
		DefaultProject: viper.GetString(keyDefaultProject),
		Timeout:        pointer.PointerOrNil(viper.GetDuration(keyTimeout)),
		Provider:       viper.GetString(keyProvider),
	}

	ctxs.Contexts = append(ctxs.Contexts, ctx)

	if viper.GetBool(keyActivate) || ctxs.CurrentContext == "" {
		ctxs.PreviousContext = ctxs.CurrentContext
		ctxs.CurrentContext = ctx.Name
	}

	err = c.cfg.WriteContexts(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, "%s added context \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.Name))

	return rq, nil
	// return nil, fmt.Errorf("testCreate")
}

func (cs *contexts) Validate() error {
	names := map[string]bool{}
	for _, context := range cs.Contexts {
		names[context.Name] = true
	}

	if len(cs.Contexts) != len(names) {
		return fmt.Errorf("context names must be unique")
	}

	return nil
}

func (c *ContextConfig) WriteContexts(ctxs *contexts) error {
	if err := ctxs.Validate(); err != nil {
		return err
	}

	fmt.Println("ctxs")
	raw, err := yaml.Marshal(ctxs)
	if err != nil {
		return err
	}

	dest, err := c.configPath()
	if err != nil {
		return err
	}

	// when path is in the default path, we ensure the directory exists
	if defaultPath, err := c.defaultConfigDirectory(); err == nil && defaultPath == path.Dir(dest) {
		err = c.Fs.MkdirAll(defaultPath, 0700)
		if err != nil {
			return fmt.Errorf("unable to ensure default config directory: %w", err)
		}
	}

	err = afero.WriteFile(c.Fs, dest, raw, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (c *ContextConfig) configPath() (string, error) {
	if viper.IsSet(keyConfig) {
		return viper.GetString(keyConfig), nil
	}

	dir, err := c.defaultConfigDirectory()
	if err != nil {
		return "", err
	}

	return path.Join(dir, c.ConfigName), nil
}

func (c *ContextConfig) defaultConfigDirectory() (string, error) {
	// TODO implement XDG specification?
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(h, "."+c.ConfigDirName), nil
}

func (c *cliWrapper) Update(rq *Context) (*Context, error) {
	return nil, fmt.Errorf("testUpdate")
}

func (c *cliWrapper) Delete(name string) (*Context, error) {
	ctxs, err := c.cfg.GetContexts()
	if err != nil {
		return nil, err
	}

	ctx, ok := ctxs.getByName(name)
	if !ok {
		return nil, fmt.Errorf("no context with name %q found", name)
	}

	ctxs.Delete(ctx.Name)

	err = c.c.WriteContexts(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, "%s removed context \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.Name))

	return nil
	return nil, fmt.Errorf("testDelete")
}

func (c *cliWrapper) Convert(r *Context) (string, *Context, *Context, error) {
	return "Yay!", &Context{}, &Context{}, fmt.Errorf("testConvert")
}

func (c *ContextConfig) GetContexts() (*contexts, error) {
	configPath, err := c.configPath()
	if err != nil {
		return nil, fmt.Errorf("cannot get config path")
	}

	raw, err := afero.ReadFile(c.Fs, configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &contexts{}, nil // TODO check consistency
		}

		return nil, fmt.Errorf("unable to read config.yaml: %w", err)
	}

	var ctxs contexts
	err = yaml.Unmarshal(raw, &ctxs)
	return &ctxs, err
}

func (cs *contexts) getByName(name string) (*Context, bool) {
	for _, context := range cs.Contexts {
		if context.Name == name {
			return context, true
		}
	}

	return nil, false
}

type tablePrinter struct{}

func (t *tablePrinter) Print(data any) error {
	ctxs, ok := data.([]*Context)
	if !ok {
		return fmt.Errorf("unsupported content")
	}
	header, rows, err := t.contextTable(ctxs, false)
	if err != nil {
		return err
	}

	fmt.Println(header)
	for _, row := range rows {
		fmt.Println(row)
	}

	return nil
}

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
				url = viper.GetString(keyApiUrl)
			}

			row = append(row, url)
		}

		rows = append(rows, row)
	}

	return header, rows, nil
}
