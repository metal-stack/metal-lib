package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"time"

	"github.com/fatih/color"
	"github.com/metal-stack/metal-lib/pkg/genericcli"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/multisort"
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
	keyWide           = "wide"

	defaultConfigName = "config.yaml"
)

var (
	// errorNotImplemented for all functions that are not implemented yet
	errorNotImplemented = fmt.Errorf("not implemented yet")
)

// Contexts contains all configuration contexts
type contexts struct {
	CurrentContext  string     `json:"current-context" yaml:"current-context"`
	PreviousContext string     `json:"previous-context" yaml:"previous-context"`
	Contexts        []*Context `json:"contexts" yaml:"contexts"`
}

type Context struct {
	Name           string         `json:"name" yaml:"name"`
	ApiURL         *string        `json:"api-url,omitempty" yaml:"api-url,omitempty"`
	Token          string         `json:"api-token" yaml:"api-token"`
	DefaultProject string         `json:"default-project" yaml:"default-project"`
	Timeout        *time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Provider       string         `json:"provider" yaml:"provider"`
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

type contextUpdateRequest struct {
	Name string
}

// NewContextCmd creates the context command tree using genericcli
func NewContextCmd(c *ContextConfig) *cobra.Command {
	// TODO check nils
	c.ConfigName = cmp.Or(c.ConfigName, string(defaultConfigName))
	c.Out = cmp.Or(c.Out, io.Writer(os.Stdout))
	c.In = cmp.Or(c.In, io.Reader(os.Stdin))
	c.Fs = cmp.Or(c.Fs, afero.NewOsFs())

	wrapper := &cliWrapper{
		cfg: c,
	}

	cmd := genericcli.NewCmds(&genericcli.CmdsConfig[
		*Context,
		*contextUpdateRequest,
		*Context,
	]{
		GenericCLI:      genericcli.NewGenericCLI(wrapper),
		BinaryName:      c.BinaryName,
		Singular:        "context",
		Plural:          "contexts",
		Description:     "Context defines the backend to talk to. Use \"-\" to switch to the previously used context.",
		Aliases:         []string{"ctx"},
		Args:            []string{keyName}, // TODO is this needed when using a flag? (--name)
		Sorter:          contextSorter(),
		DescribePrinter: c.DescribePrinter,
		ListPrinter:     func() printers.Printer { return getListPrinter(c) },
		In:              c.In,
		Out:             c.Out,
		OnlyCmds: genericcli.OnlyCmds(
			genericcli.DescribeCmd,
			genericcli.ListCmd,
			genericcli.CreateCmd,
			genericcli.UpdateCmd,
			genericcli.DeleteCmd,
		),
		RootCmdMutateFn: func(cmd *cobra.Command) {
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				// '$ BinaryName context' (no args) should be equal to '$ BinaryName context list'
				if len(args) == 0 {
					listCmd, _, err := cmd.Find([]string{"list"})
					if err != nil {
						return fmt.Errorf("internal: list command not found: %w", err)
					}
					return listCmd.RunE(listCmd, []string{})
				}

				// '$ BinaryName context -' or '$ BinaryName context <name>' should behave like 'switch'
				if len(args) == 1 {
					return c.setContext(args)
				}

				// Probably too many args, fallback to help
				return cmd.Help()
			}
		},
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

			cmd.ValidArgsFunction = c.contextListCompletion
		},
		DeleteCmdMutateFn: func(cmd *cobra.Command) {
			cmd.ValidArgsFunction = c.contextListCompletion
		},
		CreateRequestFromCLI: func() (*Context, error) {
			return &Context{}, nil // Placeholder to trigger cmdline read (not file read)
		},
		UpdateRequestFromCLI: func(args []string) (*contextUpdateRequest, error) {
			name, err := genericcli.GetExactlyOneArg(args)
			if err != nil {
				return nil, fmt.Errorf("no context name given")
			}
			return &contextUpdateRequest{Name: name}, nil
		},
	})

	switchCmd := &cobra.Command{
		Use:     "switch <context-name>",
		Short:   "switch the cli context",
		Long:    "switch the cli context. Use \"-\" to switch to the previously used context.",
		Aliases: []string{"set", "sw"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.setContext(args)
		},
		ValidArgsFunction: c.contextListCompletion,
	}

	setProjectCmd := &cobra.Command{
		Use:   "set-project <project-id>",
		Short: "sets the default project to operate on for cli commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.setProject(args)
		},
		ValidArgsFunction: c.ProjectListCompletion,
	}

	cmd.AddCommand(
		switchCmd,
		setProjectCmd,
	)

	return cmd
}

func (c *ContextConfig) setContext(args []string) error {
	wantCtx, err := genericcli.GetExactlyOneArg(args)
	if err != nil {
		return fmt.Errorf("no context name given")
	}

	ctxs, err := c.getContexts()
	if err != nil {
		return err
	}

	if wantCtx == "-" {
		if ctxs.PreviousContext == "" {
			return fmt.Errorf("no previous context found")
		}
		ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, ctxs.PreviousContext
	} else {
		if _, ok := ctxs.getByName(wantCtx); !ok {
			return fmt.Errorf("context %s not found", wantCtx)
		}
		if wantCtx == ctxs.CurrentContext {
			_, _ = fmt.Fprintf(c.Out, "%s context \"%s\" is already active\n", color.GreenString("✔"), color.GreenString(ctxs.CurrentContext))
			return nil
		}
		ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, wantCtx
	}

	err = c.writeContexts(ctxs)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.Out, "%s switched context to \"%s\"\n", color.GreenString("✔"), color.GreenString(ctxs.CurrentContext))

	return nil
}

func (c *ContextConfig) setProject(args []string) error {
	project, err := genericcli.GetExactlyOneArg(args)
	if err != nil {
		return err
	}

	ctxs, err := c.getContexts()
	if err != nil {
		return err
	}

	ctx, ok := ctxs.getByName(ctxs.CurrentContext)
	if !ok {
		return fmt.Errorf("no context currently active")
	}

	ctx.DefaultProject = project

	err = c.writeContexts(ctxs)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.Out, "%s switched context default project to \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.DefaultProject))

	return nil
}

func (c *ContextConfig) contextListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctxs, err := c.getContexts()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var names []string
	for _, ctx := range ctxs.Contexts {
		names = append(names, ctx.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func (c *ContextConfig) writeContexts(ctxs *contexts) error {
	if err := ctxs.validate(); err != nil {
		return err
	}

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

func (c *ContextConfig) getContexts() (*contexts, error) {
	configPath, err := c.configPath()
	if err != nil {
		return nil, fmt.Errorf("cannot get config path")
	}

	raw, err := afero.ReadFile(c.Fs, configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &contexts{}, nil // TODO check consistency
		}

		return nil, fmt.Errorf("unable to read %s: %w", c.ConfigName, err)
	}

	var ctxs contexts
	err = yaml.Unmarshal(raw, &ctxs)
	return &ctxs, err
}

func (c *cliWrapper) Get(name string) (*Context, error) {
	ctxs, err := c.cfg.getContexts()
	if err != nil {
		return nil, err
	}

	ctx, ok := ctxs.getByName(name)
	if !ok {
		return nil, fmt.Errorf("context %q not found", name)
	}
	return ctx, nil
}

func (c *cliWrapper) List() ([]*Context, error) {
	ctxs, err := c.cfg.getContexts()
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
	ctxs, err := c.cfg.getContexts()
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

	err = c.cfg.writeContexts(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, "%s added context \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.Name))

	return ctx, nil
}

func (c *cliWrapper) Update(rq *contextUpdateRequest) (*Context, error) {
	ctxs, err := c.cfg.getContexts()
	if err != nil {
		return nil, err
	}

	ctx, ok := ctxs.getByName(rq.Name)
	if !ok {
		return nil, fmt.Errorf("no context with name %q found", rq.Name)
	}

	if viper.IsSet(keyApiUrl) {
		ctx.ApiURL = pointer.PointerOrNil(viper.GetString(keyApiUrl))
	}
	if viper.IsSet(keyApiToken) {
		ctx.Token = viper.GetString(keyApiToken)
	}
	if viper.IsSet(keyDefaultProject) {
		ctx.DefaultProject = viper.GetString(keyDefaultProject)
	}
	if viper.IsSet(keyTimeout) {
		ctx.Timeout = pointer.PointerOrNil(viper.GetDuration(keyTimeout))
	}
	if viper.IsSet(keyProvider) {
		ctx.Provider = viper.GetString(keyProvider)
	}
	if viper.GetBool(keyActivate) {
		ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, ctx.Name
	}

	err = c.cfg.writeContexts(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, "%s updated context \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.Name))

	return ctx, nil
}

func (c *cliWrapper) Delete(name string) (*Context, error) {
	ctxs, err := c.cfg.getContexts()
	if err != nil {
		return nil, err
	}

	ctx, ok := ctxs.getByName(name)
	if !ok {
		return nil, fmt.Errorf("context %q not found", name)
	}
	// TODO Use Get ?

	ctxs.delete(ctx.Name)

	if ctxs.CurrentContext == ctx.Name {
		ctxs.CurrentContext = ""
	}

	if ctxs.PreviousContext == ctx.Name {
		ctxs.PreviousContext = ""
	}

	err = c.cfg.writeContexts(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, "%s removed context \"%s\"\n", color.GreenString("✔"), color.GreenString(ctx.Name))

	return ctx, nil
}

func (c *cliWrapper) Convert(r *Context) (string, *Context, *contextUpdateRequest, error) {
	return "", &Context{}, &contextUpdateRequest{}, errorNotImplemented // editCmd is disabled, this is not needed
}

func (cs *contexts) validate() error {
	names := map[string]bool{}
	for _, context := range cs.Contexts {
		names[context.Name] = true
	}

	if len(cs.Contexts) != len(names) {
		return fmt.Errorf("context names must be unique")
	}

	return nil
}

func (cs *contexts) delete(name string) {
	cs.Contexts = slices.DeleteFunc(cs.Contexts, func(ctx *Context) bool {
		return ctx.Name == name
	})
}

func (cs *contexts) getByName(name string) (*Context, bool) {
	for _, context := range cs.Contexts {
		if context.Name == name {
			return context, true
		}
	}
	return nil, false
}

func contextSorter() *multisort.Sorter[*Context] {
	return multisort.New(multisort.FieldMap[*Context]{
		"name": func(a, b *Context, descending bool) multisort.CompareResult {
			return multisort.Compare(a.Name, b.Name, descending)
		},
	}, multisort.Keys{{ID: "name"}})
}

func getListPrinter(c *ContextConfig) printers.Printer {
	allContexts, err := c.getContexts()
	currentContextName := ""
	if err == nil {
		currentContextName = allContexts.CurrentContext
	}

	toH := func(data any, wide bool) ([]string, [][]string, error) {
		ctxList, ok := data.([]*Context)
		if !ok {
			return nil, nil, fmt.Errorf("unsupported content: expected []*Context")
		}
		return contextTable(ctxList, wide, currentContextName)
	}

	return printers.NewTablePrinter(&printers.TablePrinterConfig{
		ToHeaderAndRows:            toH,
		Wide:                       viper.GetBool(keyWide),
		Markdown:                   false,
		NoHeaders:                  false,
		Out:                        c.Out,
		DisableDefaultErrorPrinter: false,
		DisableAutoWrap:            false,
	})
}

func contextTable(data []*Context, wide bool, currentContextName string) ([]string, [][]string, error) {
	var (
		header = []string{"", "Name", "Provider", "Default Project"}
		rows   [][]string
	)

	if wide {
		header = append(header, "API URL")
	}

	for _, c := range data {
		active := ""
		if c.Name == currentContextName {
			active = color.GreenString("✔")
		}

		row := []string{active, c.Name, c.Provider, c.DefaultProject}
		if wide {
			url := pointer.SafeDeref(c.ApiURL)
			row = append(row, url)
		}

		rows = append(rows, row)
	}

	return header, rows, nil
}
