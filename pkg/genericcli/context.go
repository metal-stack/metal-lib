package genericcli

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
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/multisort"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

const (
	keyName           = "name"
	keyAPIURL         = "api-url"
	keyAPIToken       = "api-token"
	keyDefaultProject = "default-project"
	keyProject        = "project"
	keyTimeout        = "timeout"
	keyActivate       = "activate"
	keyProvider       = "provider"
	keyConfig         = "config"

	defaultConfigName  = "config.yaml"
	DefaultContextName = "default"

	checkMarkChar = "✔"
)

// contextConfig contains all configuration contextConfig
type contextConfig struct {
	CurrentContext  string     `json:"current-context"`
	PreviousContext string     `json:"previous-context"`
	Contexts        []*Context `json:"contexts"`
}

type Context struct {
	Name           string         `json:"name"`
	APIURL         *string        `json:"api-url,omitempty"`
	APIToken       string         `json:"api-token"`
	DefaultProject string         `json:"default-project"`
	Timeout        *time.Duration `json:"timeout,omitempty"`
	Provider       string         `json:"provider"`
	IsCurrent      bool           `json:"-"`
}

type ContextCmdConfig struct {
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

type ContextManager struct {
	cfg *ContextCmdConfig
}

type ContextUpdateRequest struct {
	// Name is the ID
	Name string

	// Fields to patch
	APIURL         *string
	APIToken       *string // Pointer, even though Context.APIToken is string
	DefaultProject *string // Pointer, even though Context.DefaultProject is string
	Timeout        *time.Duration
	Provider       *string // Pointer, even though Context.Provider is string

	// Meta-flags for the operation
	IsCurrent bool
}

// NewContextCmd creates the context command tree using genericcli
func NewContextCmd(c *ContextCmdConfig) *cobra.Command {
	mgr := NewContextManager(c)

	cmd := NewCmds(&CmdsConfig[
		*Context,
		*ContextUpdateRequest,
		*Context,
	]{
		GenericCLI:      NewGenericCLI(mgr),
		BinaryName:      c.BinaryName,
		Singular:        "context",
		Plural:          "contexts",
		Description:     "Manage CLI contexts. A context defines the connection properties (API URL, token, etc.) for a backend. Use \"-\" to switch to the previously used context.",
		Aliases:         []string{"ctx"},
		Args:            []string{keyName},
		Sorter:          contextSorter(),
		DescribePrinter: c.DescribePrinter,
		ListPrinter:     c.ListPrinter,
		In:              c.In,
		Out:             c.Out,
		OnlyCmds: OnlyCmds(
			DescribeCmd,
			ListCmd,
			CreateCmd,
			UpdateCmd,
			DeleteCmd,
		),
		RootCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Args = cobra.MaximumNArgs(1)
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				// '$ BinaryName context' (no args) should be equal to '$ BinaryName context list'
				if len(args) == 0 {
					listCmd, _, err := cmd.Find(pointer.WrapInSlice(string(ListCmd)))
					if err != nil {
						return fmt.Errorf("internal: list command not found: %w", err)
					}
					return listCmd.RunE(listCmd, []string{})
				}

				// '$ BinaryName context -' or '$ BinaryName context <name>' should behave like 'switch'
				// Now we can only have one arg thanks to cobra.MaximumNArgs(1) above
				return mgr.switchContext(args)
			}
		},
		DescribeCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Args = cobra.MaximumNArgs(1)

			originalRunE := cmd.RunE

			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				// If no args are provided, try to use the current context
				if len(args) > 0 {
					return originalRunE(cmd, args)
				}

				ctxs, err := mgr.getContextConfig()
				if err != nil {
					return fmt.Errorf("unable to fetch contexts: %w", err)
				}
				if ctxs.CurrentContext == "" {
					return errors.New("no context name provided and no context is currently active")
				}

				return originalRunE(cmd, []string{ctxs.CurrentContext})
			}
		},
		ListCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Args = cobra.ExactArgs(0)
		},
		CreateCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Flags().String(keyName, "", "set the name of the context")
			cmd.Flags().String(keyAPIURL, "", "set the api-url for this context")
			cmd.Flags().String(keyAPIToken, "", "set the api-token for this context")
			cmd.Flags().String(keyDefaultProject, "", "set a default project to operate on")
			cmd.Flags().Duration(keyTimeout, 0, "set a default request timeout")
			cmd.Flags().Bool(keyActivate, false, "immediately switches to the new context")
			cmd.Flags().String(keyProvider, "", "set the login provider for this context")

			Must(cmd.MarkFlagRequired(keyName))
			Must(cmd.MarkFlagRequired(keyAPIToken))

			cmd.Args = cobra.ExactArgs(0)
		},
		UpdateCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Flags().String(keyAPIURL, "", "set the api-url for this context")
			cmd.Flags().String(keyAPIToken, "", "set the api-token for this context")
			cmd.Flags().String(keyDefaultProject, "", "set a default project to operate on")
			cmd.Flags().Duration(keyTimeout, 0, "set a default request timeout")
			cmd.Flags().Bool(keyActivate, false, "immediately switches to the new context")
			cmd.Flags().String(keyProvider, "", "set the login provider for this context")

			Must(cmd.RegisterFlagCompletionFunc(keyDefaultProject, c.ProjectListCompletion))

			cmd.ValidArgsFunction = mgr.ContextListCompletion

			cmd.Args = cobra.ExactArgs(1)
		},
		DeleteCmdMutateFn: func(cmd *cobra.Command) {
			cmd.ValidArgsFunction = mgr.ContextListCompletion

			cmd.Args = cobra.ExactArgs(1)
		},
		CreateRequestFromCLI: func() (*Context, error) {
			name := viper.GetString(keyName)

			ctx := &Context{
				Name:           name,
				APIURL:         pointer.PointerOrNil(viper.GetString(keyAPIURL)),
				APIToken:       viper.GetString(keyAPIToken),
				DefaultProject: viper.GetString(keyDefaultProject),
				Timeout:        pointer.PointerOrNil(viper.GetDuration(keyTimeout)),
				Provider:       viper.GetString(keyProvider),
				IsCurrent:      viper.GetBool(keyActivate),
			}
			return ctx, nil
		},
		UpdateRequestFromCLI: func(args []string) (*ContextUpdateRequest, error) {
			name, err := GetExactlyOneArg(args)
			if err != nil {
				return nil, err
			}

			return &ContextUpdateRequest{
				Name:           name,
				IsCurrent:      viper.GetBool(keyActivate),
				APIURL:         getFromViper(keyAPIURL, viper.GetString),
				APIToken:       getFromViper(keyAPIToken, viper.GetString),
				DefaultProject: getFromViper(keyDefaultProject, viper.GetString),
				Timeout:        getFromViper(keyTimeout, viper.GetDuration),
				Provider:       getFromViper(keyProvider, viper.GetString),
			}, nil
		},
	})

	switchCmd := &cobra.Command{
		Use:     "switch <context-name>",
		Short:   "switch the active CLI context",
		Long:    "Switch the active CLI context. Use \"-\" to switch to the previously used context.",
		Aliases: []string{"set", "sw"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mgr.switchContext(args)
		},
		ValidArgsFunction: mgr.ContextListCompletion,
	}

	setProjectCmd := &cobra.Command{
		Use:   "set-project <project-id>",
		Args:  cobra.ExactArgs(1),
		Short: "set the default project to operate on for cli commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mgr.setProject(args)
		},
		ValidArgsFunction: c.ProjectListCompletion,
	}

	showCurrentCmd := &cobra.Command{
		Use:   "show-current",
		Short: "print the active context name",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctxs, err := mgr.getContextConfig()
			if err != nil {
				return fmt.Errorf("unable to fetch contexts: %w", err)
			}
			if ctxs.CurrentContext == "" {
				return errors.New("no context currently active")
			}

			_, err = fmt.Fprint(c.Out, ctxs.CurrentContext)
			return err
		},
		ValidArgsFunction: mgr.ContextListCompletion,
	}

	cmd.AddCommand(
		switchCmd,
		setProjectCmd,
		showCurrentCmd,
	)

	return cmd
}

func NewContextManager(c *ContextCmdConfig) *ContextManager {
	c.ConfigName = cmp.Or(c.ConfigName, string(defaultConfigName))
	c.Out = cmp.Or(c.Out, io.Writer(os.Stdout))
	c.In = cmp.Or(c.In, io.Reader(os.Stdin))
	c.Fs = cmp.Or(c.Fs, afero.NewOsFs())

	if c.BinaryName == "" {
		panic(fmt.Errorf("ContextConfig has a required option \"%s\" missing", "BinaryName"))
	}

	if c.ConfigDirName == "" {
		panic(errors.New("no config directory name provided"))
	}

	if c.ListPrinter == nil {
		panic(fmt.Errorf("ContextConfig has a required option \"%s\" missing", "ListPrinter"))
	}

	if c.DescribePrinter == nil {
		panic(fmt.Errorf("ContextConfig has a required option \"%s\" missing", "DescribePrinter"))
	}

	// ProjectListCompletion is not crucial so we skip the check

	return &ContextManager{cfg: c}
}

// ContextTable returns the table representation of a list of contexts. Used by printers in CLI implementations
func ContextTable(data any, wide bool) ([]string, [][]string, error) {
	ctxList, ok := data.([]*Context)
	if !ok {
		return nil, nil, errors.New("unsupported content: expected []*Context")
	}

	if len(ctxList) == 0 {
		return nil, nil, errors.New("you need to create a context first")
	}

	var (
		header = []string{"", "Name", "Provider", "Default Project"}
		rows   [][]string
	)

	if wide {
		header = append(header, "API URL")
	}

	for _, c := range ctxList {
		active := ""
		if c.IsCurrent {
			active = color.GreenString(checkMarkChar)
		}

		row := []string{active, c.Name, c.Provider, c.DefaultProject}
		if wide {
			url := pointer.SafeDeref(c.APIURL)
			row = append(row, url)
		}

		rows = append(rows, row)
	}

	return header, rows, nil
}

func (c *ContextManager) Get(name string) (*Context, error) {
	ctxs, err := c.getContextConfig()
	if err != nil {
		return nil, err
	}

	ctx, ok := ctxs.getByName(name)
	if !ok {
		return nil, fmt.Errorf("context \"%s\" not found", name)
	}
	return ctx, nil
}

func (c *ContextManager) List() ([]*Context, error) {
	ctxs, err := c.getContextConfig()
	if err != nil {
		return nil, err
	}

	return ctxs.Contexts, nil
}

func (c *ContextManager) Create(rq *Context) (*Context, error) {
	ctxs, err := c.getContextConfig()
	if err != nil {
		return nil, err
	}

	ctxs.Contexts = append(ctxs.Contexts, rq)

	if rq.IsCurrent || ctxs.CurrentContext == "" {
		ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, rq.Name
		rq.IsCurrent = true // this is needed to return the right context state
	}

	// name uniqness check is performed by writeContexts
	err = c.writeContextConfig(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, "%s Added context \"%s\"\n", color.GreenString(checkMarkChar), color.GreenString(rq.Name))

	return rq, nil
}

func (c *ContextManager) Update(rq *ContextUpdateRequest) (*Context, error) {
	ctxs, err := c.getContextConfig()
	if err != nil {
		return nil, err
	}

	switch rq.Name {
	case "": // defaults to current context if no name is provided
		if ctxs.CurrentContext == "" {
			return nil, errors.New("no context currently active")
		}
		rq.Name = ctxs.CurrentContext
	case "-":
		if ctxs.PreviousContext == "" {
			return nil, errors.New("no previous context found")
		}
		rq.Name = ctxs.PreviousContext
	}

	ctx, ok := ctxs.getByName(rq.Name)
	if !ok {
		return nil, fmt.Errorf("context \"%s\" not found", rq.Name)
	}

	var updated bool

	if rq.APIURL != nil {
		ctx.APIURL = rq.APIURL
		updated = true
	}
	if rq.APIToken != nil {
		ctx.APIToken = *rq.APIToken
		updated = true
	}
	if rq.DefaultProject != nil {
		ctx.DefaultProject = *rq.DefaultProject
		updated = true
	}
	if rq.Timeout != nil {
		ctx.Timeout = rq.Timeout
		updated = true
	}
	if rq.Provider != nil {
		ctx.Provider = *rq.Provider
		updated = true
	}

	var switched bool
	if rq.IsCurrent && ctxs.CurrentContext != rq.Name {
		ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, rq.Name
		switched = true
	}

	err = c.writeContextConfig(ctxs)
	if err != nil {
		return nil, err
	}

	if updated {
		_, _ = fmt.Fprintf(c.cfg.Out, "%s Updated context \"%s\"\n", color.GreenString(checkMarkChar), color.GreenString(rq.Name))
	}
	if switched {
		_, _ = fmt.Fprintf(c.cfg.Out, "%s Switched context to \"%s\"\n", color.GreenString(checkMarkChar), color.GreenString(ctxs.CurrentContext))
	} else if rq.IsCurrent {
		_, _ = fmt.Fprintf(c.cfg.Out, "%s Context \"%s\" is already active\n", color.GreenString(checkMarkChar), color.GreenString(ctxs.CurrentContext))
	}

	return ctx, nil
}

func (c *ContextManager) Delete(name string) (*Context, error) {
	ctxs, err := c.getContextConfig()
	if err != nil {
		return nil, err
	}

	deletedCtx := ctxs.remove(name)
	if deletedCtx == nil {
		return nil, fmt.Errorf("context \"%s\" not found", name)
	}

	if ctxs.CurrentContext == name {
		ctxs.CurrentContext = ""
	}

	if ctxs.PreviousContext == name {
		ctxs.PreviousContext = ""
	}

	err = c.writeContextConfig(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, "%s Removed context \"%s\"\n", color.GreenString(checkMarkChar), color.GreenString(name))

	return deletedCtx, nil
}

// Convert is not used as editCmd is disabled
func (c *ContextManager) Convert(r *Context) (string, *Context, *ContextUpdateRequest, error) {
	return r.Name, r, &ContextUpdateRequest{
		Name:           r.Name,
		APIURL:         r.APIURL,
		APIToken:       &r.APIToken,
		DefaultProject: &r.DefaultProject,
		Timeout:        r.Timeout,
		Provider:       &r.Provider,
		IsCurrent:      r.IsCurrent,
	}, nil
}

func (c *ContextManager) ContextListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctxs, err := c.getContextConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var names []string
	for _, ctx := range ctxs.Contexts {
		names = append(names, ctx.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func (c *ContextManager) switchContext(args []string) error {
	wantCtxName, err := GetExactlyOneArg(args)
	if err != nil {
		return err
	}

	_, err = c.Update(&ContextUpdateRequest{
		Name:      wantCtxName,
		IsCurrent: true,
	})
	return err
}

func (c *ContextManager) setProject(args []string) error {
	project, err := GetExactlyOneArg(args)
	if err != nil {
		return err
	}

	_, err = c.Update(&ContextUpdateRequest{DefaultProject: &project})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, "%s Switched context default project to \"%s\"\n", color.GreenString(checkMarkChar), color.GreenString(project))

	return nil
}

func (c *ContextManager) writeContextConfig(ctxs *contextConfig) error {
	if err := ctxs.validate(); err != nil {
		return err
	}
	raw, err := yaml.Marshal(ctxs)
	if err != nil {
		return err
	}

	dest, err := c.cfg.configPath()
	if err != nil {
		return err
	}

	defaultPath, err := c.cfg.defaultConfigDirectory()
	if err != nil {
		return fmt.Errorf("failed to get default config directory: %w", err)
	}
	if defaultPath == path.Dir(dest) {
		err = c.cfg.Fs.MkdirAll(defaultPath, 0700)
		if err != nil {
			return fmt.Errorf("unable to ensure default config directory: %w", err)
		}
	}

	err = afero.WriteFile(c.cfg.Fs, dest, raw, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (c *ContextManager) getContextConfig() (*contextConfig, error) {
	configPath, err := c.cfg.configPath()
	if err != nil {
		return nil, fmt.Errorf("unable to determine config path: %w", err)
	}

	raw, err := afero.ReadFile(c.cfg.Fs, configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &contextConfig{
				Contexts: []*Context{},
			}, nil
		}

		return nil, fmt.Errorf("unable to read %s: %w", c.cfg.ConfigName, err)
	}

	var ctxs contextConfig
	err = yaml.Unmarshal(raw, &ctxs)

	if ctxCurrent, ok := ctxs.getByName(ctxs.CurrentContext); ok {
		ctxCurrent.IsCurrent = true
	}

	return &ctxs, err
}

func (c *Context) GetProject() string {
	if viper.IsSet(keyProject) {
		return viper.GetString(keyProject)
	}
	return c.DefaultProject
}

func (c *Context) GetAPIToken() string {
	// TODO ensure consistency
	if viper.IsSet(keyAPIToken) {
		return viper.GetString(keyAPIToken)
	}
	return c.APIToken
}

func (c *Context) GetAPIURL() string {
	if c.APIURL != nil {
		return *c.APIURL
	}

	// fallback to the default specified by viper
	return viper.GetString(keyAPIURL)
}

func (c *Context) GetProvider() string {
	if viper.IsSet(keyProvider) {
		return viper.GetString(keyProvider)
	}
	return c.Provider
}

func (cs *contextConfig) validate() error {
	names := map[string]bool{}
	for _, context := range cs.Contexts {
		names[context.Name] = true
		if context.Name == "" {
			return fmt.Errorf("context field \"%s\" cannot be blank", "Name")
		}
		if context.APIToken == "" {
			return fmt.Errorf("context field \"%s\" cannot be blank", "APIToken")
		}
	}

	if len(cs.Contexts) != len(names) {
		return errors.New("context names must be unique")
	}

	return nil
}

func (cs *contextConfig) remove(name string) *Context {
	var deletedCtx *Context
	cs.Contexts = slices.DeleteFunc(cs.Contexts, func(ctx *Context) bool {
		if ctx.Name == name {
			deletedCtx = ctx
		}
		return ctx.Name == name
	})

	return deletedCtx
}

func (cs *contextConfig) getByName(name string) (*Context, bool) {
	for _, context := range cs.Contexts {
		if context.Name == name {
			return context, true
		}
	}
	return nil, false
}

func (c *ContextManager) GetContextCurrentOrDefault() *Context {
	ctxs, err := c.getContextConfig()
	if err != nil {
		return defaultCtx()
	}
	ctx, ok := ctxs.getByName(ctxs.CurrentContext)
	if !ok {
		return defaultCtx()
	}
	return ctx
}

func (c *ContextManager) GetCurrentContext() (*Context, error) {
	ctxList, err := c.List()
	if err != nil {
		return nil, err
	}

	for _, ctx := range ctxList {
		if ctx.IsCurrent {
			return ctx, nil
		}
	}
	return nil, nil
}

func defaultCtx() *Context {
	return &Context{
		Name:     DefaultContextName,
		APIURL:   pointer.PointerOrNil(viper.GetString(keyAPIURL)),
		APIToken: viper.GetString(keyAPIToken),
	}
}

func (c *ContextCmdConfig) configPath() (string, error) {
	if viper.IsSet(keyConfig) {
		return viper.GetString(keyConfig), nil
	}

	dir, err := c.defaultConfigDirectory()
	if err != nil {
		return "", err
	}

	return path.Join(dir, c.ConfigName), nil
}

func (c *ContextCmdConfig) defaultConfigDirectory() (string, error) {
	// TODO implement XDG specification?
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(h, "."+c.ConfigDirName), nil
}

func contextSorter() *multisort.Sorter[*Context] {
	return multisort.New(multisort.FieldMap[*Context]{
		keyName: func(a, b *Context, descending bool) multisort.CompareResult {
			return multisort.Compare(a.Name, b.Name, descending)
		},
		keyAPIURL: func(a, b *Context, descending bool) multisort.CompareResult {
			urlA := pointer.SafeDeref(a.APIURL)
			urlB := pointer.SafeDeref(b.APIURL)
			return multisort.Compare(urlA, urlB, descending)
		},
		keyDefaultProject: func(a, b *Context, descending bool) multisort.CompareResult {
			return multisort.Compare(a.DefaultProject, b.DefaultProject, descending)
		},
		keyTimeout: func(a, b *Context, descending bool) multisort.CompareResult {
			timeoutA := pointer.SafeDeref(a.Timeout)
			timeoutB := pointer.SafeDeref(b.Timeout)
			return multisort.Compare(timeoutA, timeoutB, descending)
		},
		keyProvider: func(a, b *Context, descending bool) multisort.CompareResult {
			return multisort.Compare(a.Provider, b.Provider, descending)
		},
	}, multisort.Keys{{ID: keyName}})
}

// getFromViper is a helper function to set ContextUpdateRequest fields
func getFromViper[T any](key string, getFunc func(string) T) *T {
	if viper.IsSet(key) {
		return pointer.Pointer(getFunc(key))
	}
	return nil
}
