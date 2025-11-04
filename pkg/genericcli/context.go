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
	"gopkg.in/yaml.v3"
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
	keyContextName    = "context"

	sortKeyName           = keyName
	sortKeyAPIURL         = keyAPIURL
	sortKeyDefaultProject = keyDefaultProject
	sortKeyTimeout        = keyTimeout
	sortKeyProvider       = keyProvider

	defaultConfigName = "config.yaml"

	// Success message formats
	msgContextAlreadyActive = "%s Context \"%s\" is already active\n"
	msgContextSwitched      = "%s Switched context to \"%s\"\n"
	msgContextUpdated       = "%s Updated context \"%s\"\n"
	msgContextAdded         = "%s Added context \"%s\"\n"
	msgContextRemoved       = "%s Removed context \"%s\"\n"
	msgProjectSwitched      = "%s Switched context default project to \"%s\"\n"

	// Error message formats
	errMsgContextNotFound         = "context \"%s\" not found"
	errMsgGettingDefaultDirFailed = "failed to get default config directory: %w"
	errMsgCannotEnsureDefaultDir  = "unable to ensure default config directory: %w"
	errMsgCannotGetConfigPath     = "unable to determine config path: %w"
	errMsgCannotReadConfig        = "unable to read %s: %w"
	errMsgCannotFetchContexts     = "unable to fetch contexts: %w"
)

var (
	// errorNotImplemented for all functions that are not implemented yet
	errorNotImplemented = fmt.Errorf("not implemented yet")
)

// contexts contains all configuration contexts
type contexts struct {
	CurrentContext  string     `json:"current-context" yaml:"current-context"`
	PreviousContext string     `json:"previous-context" yaml:"previous-context"`
	Contexts        []*Context `json:"contexts" yaml:"contexts"`
}

type Context struct {
	Name           string         `json:"name" yaml:"name"`
	APIURL         *string        `json:"api-url,omitempty" yaml:"api-url,omitempty"`
	APIToken       string         `json:"api-token" yaml:"api-token"`
	DefaultProject string         `json:"default-project" yaml:"default-project"`
	Timeout        *time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Provider       string         `json:"provider" yaml:"provider"`
	IsCurrent      bool           `json:"-" yaml:"-"`
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
	// Name is the ID
	Name string

	// Fields to patch
	APIURL         *string
	APIToken       *string // Pointer, even though Context.APIToken is string
	DefaultProject *string // Pointer, even though Context.DefaultProject is string
	Timeout        *time.Duration
	Provider       *string // Pointer, even though Context.Provider is string

	// Meta-flags for the operation
	Activate bool
}

// setFromViper is a helper function to set contextUpdateRequest fields
func setFromViper[T any](key string, getFunc func(string) T) *T {
	if viper.IsSet(key) {
		return pointer.Pointer(getFunc(key))
	}
	return nil
}

// successCheck returns a green checkmark string.
func successCheck() string {
	return color.GreenString("✔")
}

// NewContextCmd creates the context command tree using genericcli
func NewContextCmd(c *ContextConfig) *cobra.Command {
	// TODO check nils
	c.ConfigName = cmp.Or(c.ConfigName, string(defaultConfigName))
	c.Out = cmp.Or(c.Out, io.Writer(os.Stdout))
	c.In = cmp.Or(c.In, io.Reader(os.Stdin))
	c.Fs = cmp.Or(c.Fs, afero.NewOsFs())

	if c.ConfigDirName == "" {
		panic(fmt.Errorf("no config directory name provided"))
	}

	wrapper := &cliWrapper{
		cfg: c,
	}

	cmd := NewCmds(&CmdsConfig[
		*Context,
		*contextUpdateRequest,
		*Context,
	]{
		GenericCLI:      NewGenericCLI(wrapper),
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
					listCmd, _, err := cmd.Find([]string{"list"})
					if err != nil {
						return fmt.Errorf("internal: list command not found: %w", err)
					}
					return listCmd.RunE(listCmd, []string{})
				}

				// '$ BinaryName context -' or '$ BinaryName context <name>' should behave like 'switch'
				// Now we can only have one arg thanks to cobra.MaximumNArgs(1) above
				return wrapper.switchContext(args)
			}
		},
		DescribeCmdMutateFn: func(cmd *cobra.Command) {
			cmd.Args = cobra.MaximumNArgs(1)

			originalRunE := cmd.RunE

			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				// If no args are provided, try to use the current context
				if len(args) == 0 {
					ctxs, err := wrapper.getContexts()
					if err != nil {
						return fmt.Errorf(errMsgCannotFetchContexts, err)
					}
					if ctxs.CurrentContext == "" {
						return fmt.Errorf("no context name provided and no context is currently active")
					}
					args = []string{ctxs.CurrentContext}
				}
				return originalRunE(cmd, args)
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

			cmd.ValidArgsFunction = wrapper.contextListCompletion

			cmd.Args = cobra.ExactArgs(1)
		},
		DeleteCmdMutateFn: func(cmd *cobra.Command) {
			cmd.ValidArgsFunction = wrapper.contextListCompletion

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
		UpdateRequestFromCLI: func(args []string) (*contextUpdateRequest, error) {
			name, err := GetExactlyOneArg(args)
			if err != nil {
				return nil, err
			}

			return &contextUpdateRequest{
				Name:           name,
				Activate:       viper.GetBool(keyActivate),
				APIURL:         setFromViper(keyAPIURL, viper.GetString),
				APIToken:       setFromViper(keyAPIToken, viper.GetString),
				DefaultProject: setFromViper(keyDefaultProject, viper.GetString),
				Timeout:        setFromViper(keyTimeout, viper.GetDuration),
				Provider:       setFromViper(keyProvider, viper.GetString),
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
			return wrapper.switchContext(args)
		},
		ValidArgsFunction: wrapper.contextListCompletion,
	}

	setProjectCmd := &cobra.Command{
		Use:   "set-project <project-id>",
		Args:  cobra.ExactArgs(1),
		Short: "set the default project to operate on for cli commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return wrapper.setProject(args)
		},
		ValidArgsFunction: c.ProjectListCompletion,
	}

	showCurrentCmd := &cobra.Command{
		Use:   "show-current",
		Short: "print the active context name",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctxs, err := wrapper.getContexts()
			if err != nil {
				return fmt.Errorf(errMsgCannotFetchContexts, err)
			}
			if ctxs.CurrentContext == "" {
				return fmt.Errorf("no context currently active")
			}

			_, err = fmt.Fprint(c.Out, ctxs.CurrentContext)
			return err
		},
		ValidArgsFunction: wrapper.contextListCompletion,
	}

	cmd.AddCommand(
		switchCmd,
		setProjectCmd,
		showCurrentCmd,
	)

	return cmd
}

func (c *cliWrapper) switchContext(args []string) error {
	wantCtxName, err := GetExactlyOneArg(args)
	if err != nil {
		return err
	}

	ctxs, err := c.getContexts()
	if err != nil {
		return err
	}

	if wantCtxName == ctxs.CurrentContext {
		_, _ = fmt.Fprintf(c.cfg.Out, msgContextAlreadyActive, successCheck(), color.GreenString(ctxs.CurrentContext))
		return nil
	}

	if wantCtxName == "-" {
		if ctxs.PreviousContext == "" {
			return fmt.Errorf("no previous context found")
		}
		wantCtxName = ctxs.PreviousContext
	} else if _, ok := ctxs.getByName(wantCtxName); !ok {
		return fmt.Errorf(errMsgContextNotFound, wantCtxName)
	}

	ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, wantCtxName

	err = c.writeContexts(ctxs)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, msgContextSwitched, successCheck(), color.GreenString(ctxs.CurrentContext))

	return nil
}

func (c *cliWrapper) setProject(args []string) error {
	project, err := GetExactlyOneArg(args)
	if err != nil {
		return err
	}

	_, err = c.Update(&contextUpdateRequest{DefaultProject: &project})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, msgProjectSwitched, successCheck(), color.GreenString(project))

	return nil
}

func (c *cliWrapper) contextListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

func (c *cliWrapper) writeContexts(ctxs *contexts) error {
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

	// when path is in the default path, we ensure the directory exists
	defaultPath, err := c.cfg.defaultConfigDirectory()
	if err != nil {
		return fmt.Errorf(errMsgGettingDefaultDirFailed, err)
	}
	if defaultPath == path.Dir(dest) {
		err = c.cfg.Fs.MkdirAll(defaultPath, 0700)
		if err != nil {
			return fmt.Errorf(errMsgCannotEnsureDefaultDir, err)
		}
	}

	err = afero.WriteFile(c.cfg.Fs, dest, raw, 0600)
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

func (c *cliWrapper) getContexts() (*contexts, error) {
	configPath, err := c.cfg.configPath()
	if err != nil {
		return nil, fmt.Errorf(errMsgCannotGetConfigPath, err)
	}

	raw, err := afero.ReadFile(c.cfg.Fs, configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &contexts{}, nil
		}

		return nil, fmt.Errorf(errMsgCannotReadConfig, c.cfg.ConfigName, err)
	}

	var ctxs contexts
	err = yaml.Unmarshal(raw, &ctxs)

	if ctxCurrent, ok := ctxs.getByName(ctxs.CurrentContext); ok {
		ctxCurrent.IsCurrent = true
	}

	return &ctxs, err
}

func (c *cliWrapper) Get(name string) (*Context, error) {
	ctxs, err := c.getContexts()
	if err != nil {
		return nil, err
	}

	ctx, ok := ctxs.getByName(name)
	if !ok {
		return nil, fmt.Errorf(errMsgContextNotFound, name)
	}
	return ctx, nil
}

func (c *cliWrapper) List() ([]*Context, error) {
	ctxs, err := c.getContexts()
	if err != nil {
		return nil, err
	}

	return ctxs.Contexts, nil
}

func (c *cliWrapper) Create(rq *Context) (*Context, error) {
	ctxs, err := c.getContexts()
	if err != nil {
		return nil, err
	}

	ctxs.Contexts = append(ctxs.Contexts, rq)

	if rq.IsCurrent || ctxs.CurrentContext == "" {
		ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, rq.Name
	}

	// name uniqness check is performed by writeContexts
	err = c.writeContexts(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, msgContextAdded, successCheck(), color.GreenString(rq.Name))

	return rq, nil
}

func (c *cliWrapper) Update(rq *contextUpdateRequest) (*Context, error) {
	ctxs, err := c.getContexts()
	if err != nil {
		return nil, err
	}

	if rq.Name == "" { // defaults to current context if no name is provided
		if ctxs.CurrentContext == "" {
			return nil, fmt.Errorf("no context currently active")
		}
		rq.Name = ctxs.CurrentContext
	}

	ctx, ok := ctxs.getByName(ctxs.CurrentContext)
	if !ok {
		return nil, fmt.Errorf(errMsgContextNotFound, rq.Name)
	}

	if rq.APIURL != nil {
		ctx.APIURL = rq.APIURL
	}
	if rq.APIToken != nil {
		ctx.APIToken = *rq.APIToken
	}
	if rq.DefaultProject != nil {
		ctx.DefaultProject = *rq.DefaultProject
	}
	if rq.Timeout != nil {
		ctx.Timeout = rq.Timeout
	}
	if rq.Provider != nil {
		ctx.Provider = *rq.Provider
	}

	var switched bool
	if rq.Activate && ctxs.CurrentContext != rq.Name {
		ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, rq.Name
		switched = true
	}

	err = c.writeContexts(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, msgContextUpdated, successCheck(), color.GreenString(rq.Name))
	if switched {
		_, _ = fmt.Fprintf(c.cfg.Out, msgContextSwitched, successCheck(), color.GreenString(ctxs.CurrentContext))
	} else if rq.Activate {
		_, _ = fmt.Fprintf(c.cfg.Out, msgContextAlreadyActive, successCheck(), color.GreenString(ctxs.CurrentContext))
	}

	return ctx, nil
}

func (c *cliWrapper) Delete(name string) (*Context, error) {
	ctxs, err := c.getContexts()
	if err != nil {
		return nil, err
	}

	deletedCtx := ctxs.delete(name)
	if deletedCtx == nil {
		return nil, fmt.Errorf(errMsgContextNotFound, name)
	}

	if ctxs.CurrentContext == name {
		ctxs.CurrentContext = ""
	}

	if ctxs.PreviousContext == name {
		ctxs.PreviousContext = ""
	}

	// TODO We don't need to call syncCurrentAndPreviousFlags here, right? The operation is over and metalctl should just exit
	err = c.writeContexts(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, msgContextRemoved, successCheck(), color.GreenString(name))

	return deletedCtx, nil
}

func (c *cliWrapper) Convert(r *Context) (string, *Context, *contextUpdateRequest, error) {
	return "", &Context{}, &contextUpdateRequest{}, errorNotImplemented // editCmd is disabled, this is not needed
}

func (c *Context) GetProject() string {
	if viper.IsSet(keyProject) {
		return viper.GetString(keyProject)
	}
	return c.DefaultProject
}

func (c *Context) GetAPIToken() string {
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
	// TODO why is it a default? Taken from metalctlv2
	return viper.GetString(keyAPIURL)
}

func (c *Context) GetProvider() string {
	if viper.IsSet(keyProvider) {
		return viper.GetString(keyProvider)
	}
	return c.Provider
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

func (cs *contexts) delete(name string) *Context {
	var deletedCtx *Context
	cs.Contexts = slices.DeleteFunc(cs.Contexts, func(ctx *Context) bool {
		if ctx.Name == name {
			deletedCtx = ctx
		}
		return ctx.Name == name
	})

	return deletedCtx
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
		sortKeyName: func(a, b *Context, descending bool) multisort.CompareResult {
			return multisort.Compare(a.Name, b.Name, descending)
		},
		sortKeyAPIURL: func(a, b *Context, descending bool) multisort.CompareResult {
			urlA := pointer.SafeDeref(a.APIURL)
			urlB := pointer.SafeDeref(b.APIURL)
			return multisort.Compare(urlA, urlB, descending)
		},
		sortKeyDefaultProject: func(a, b *Context, descending bool) multisort.CompareResult {
			return multisort.Compare(a.DefaultProject, b.DefaultProject, descending)
		},
		sortKeyTimeout: func(a, b *Context, descending bool) multisort.CompareResult {
			timeoutA := pointer.SafeDeref(a.Timeout)
			timeoutB := pointer.SafeDeref(b.Timeout)
			return multisort.Compare(timeoutA, timeoutB, descending)
		},
		sortKeyProvider: func(a, b *Context, descending bool) multisort.CompareResult {
			return multisort.Compare(a.Provider, b.Provider, descending)
		},
	}, multisort.Keys{{ID: sortKeyName}})
}

func (c *cliWrapper) MustDefaultContext() Context {
	ctxs, err := c.getContexts()
	if err != nil {
		return defaultCtx()
	}
	ctx, ok := ctxs.getByName(ctxs.CurrentContext)
	if !ok {
		return defaultCtx()
	}
	return *ctx
}

func defaultCtx() Context {
	return Context{
		APIURL:   pointer.PointerOrNil(viper.GetString(keyAPIURL)),
		APIToken: viper.GetString(keyAPIToken),
	}
}

func ContextTable(data any, wide bool) ([]string, [][]string, error) {
	ctxList, ok := data.([]*Context)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported content: expected []*Context")
	}

	if len(ctxList) == 0 {
		return nil, nil, fmt.Errorf("you need to create a context first")
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
			active = successCheck()
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
