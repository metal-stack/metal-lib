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
	keyContextName    = "context"

	sortKeyName           = keyName
	sortKeyAPIURL         = keyAPIURL
	sortKeyDefaultProject = keyDefaultProject
	sortKeyTimeout        = keyTimeout
	sortKeyProvider       = keyProvider

	defaultConfigName  = "config.yaml"
	DefaultContextName = "default"

	// Success message formats
	msgContextAlreadyActive = "%s Context \"%s\" is already active\n"
	msgContextSwitched      = "%s Switched context to \"%s\"\n"
	msgContextUpdated       = "%s Updated context \"%s\"\n"
	msgContextAdded         = "%s Added context \"%s\"\n"
	msgContextRemoved       = "%s Removed context \"%s\"\n"
	msgProjectSwitched      = "%s Switched context default project to \"%s\"\n"

	// Error message formats
	errMsgContextNotFound         = "context \"%s\" not found"
	errMsgCannotGetDefaultDir     = "failed to get default config directory: %w"
	errMsgCannotEnsureDefaultDir  = "unable to ensure default config directory: %w"
	errMsgCannotGetConfigPath     = "unable to determine config path: %w"
	errMsgCannotReadConfig        = "unable to read %s: %w"
	errMsgCannotFetchContexts     = "unable to fetch contexts: %w"
	errMsgCannotWriteContexts     = "failed to save contexts: %w"
	errMsgListCommandNotFound     = "internal: list command not found: %w"
	errMsgContextConfigIncomplete = "ContextConfig has a required option \"%s\" missing"
	errMsgBlankContextField       = "context field \"%s\" cannot be blank"
)

var (
	errNoConfigDirName        = errors.New("no config directory name provided")
	errNoContextGivenOrActive = errors.New("no context name provided and no context is currently active")
	errNoActiveContext        = errors.New("no context currently active")
	errNoPreviousContext      = errors.New("no previous context found")
	errContextNamesAreUnique  = errors.New("context names must be unique")
	errExpectedContextSlice   = errors.New("unsupported content: expected []*Context")
	errCreateContextFirst     = errors.New("you need to create a context first")
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

type ContextManagerConfig struct {
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
	cfg *ContextManagerConfig
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
	Activate bool
}

// getFromViper is a helper function to set ContextUpdateRequest fields
func getFromViper[T any](key string, getFunc func(string) T) *T {
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
func NewContextCmd(c *ContextManagerConfig) *cobra.Command {
	wrapper := NewContextManager(c)

	cmd := NewCmds(&CmdsConfig[
		*Context,
		*ContextUpdateRequest,
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
					listCmd, _, err := cmd.Find(pointer.WrapInSlice(string(ListCmd)))
					if err != nil {
						return fmt.Errorf(errMsgListCommandNotFound, err)
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
						return errNoContextGivenOrActive
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

			cmd.ValidArgsFunction = wrapper.ContextListCompletion

			cmd.Args = cobra.ExactArgs(1)
		},
		DeleteCmdMutateFn: func(cmd *cobra.Command) {
			cmd.ValidArgsFunction = wrapper.ContextListCompletion

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
				Activate:       viper.GetBool(keyActivate),
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
			return wrapper.switchContext(args)
		},
		ValidArgsFunction: wrapper.ContextListCompletion,
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
				return errNoActiveContext
			}

			_, err = fmt.Fprint(c.Out, ctxs.CurrentContext)
			return err
		},
		ValidArgsFunction: wrapper.ContextListCompletion,
	}

	cmd.AddCommand(
		switchCmd,
		setProjectCmd,
		showCurrentCmd,
	)

	return cmd
}

func NewContextManager(c *ContextManagerConfig) *ContextManager {
	c.ConfigName = cmp.Or(c.ConfigName, string(defaultConfigName))
	c.Out = cmp.Or(c.Out, io.Writer(os.Stdout))
	c.In = cmp.Or(c.In, io.Reader(os.Stdin))
	c.Fs = cmp.Or(c.Fs, afero.NewOsFs())

	if c.BinaryName == "" {
		panic(fmt.Errorf(errMsgContextConfigIncomplete, "BinaryName"))
	}

	if c.ConfigDirName == "" {
		panic(errNoConfigDirName)
	}

	if c.ListPrinter == nil {
		panic(fmt.Errorf(errMsgContextConfigIncomplete, "ListPrinter"))
	}

	if c.DescribePrinter == nil {
		panic(fmt.Errorf(errMsgContextConfigIncomplete, "DescribePrinter"))
	}

	// ProjectListCompletion is not crucial so we skip the check

	return &ContextManager{cfg: c}
}

func (c *ContextManager) switchContext(args []string) error {
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
			return errNoPreviousContext
		}
		wantCtxName = ctxs.PreviousContext
	} else if _, ok := ctxs.getByName(wantCtxName); !ok {
		return fmt.Errorf(errMsgContextNotFound, wantCtxName)
	}

	ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, wantCtxName

	err = c.writeContextConfig(ctxs)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, msgContextSwitched, successCheck(), color.GreenString(ctxs.CurrentContext))

	return nil
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

	_, _ = fmt.Fprintf(c.cfg.Out, msgProjectSwitched, successCheck(), color.GreenString(project))

	return nil
}

func (c *ContextManager) ContextListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

	// when path is in the default path, we ensure the directory exists
	defaultPath, err := c.cfg.defaultConfigDirectory()
	if err != nil {
		return fmt.Errorf(errMsgCannotGetDefaultDir, err)
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

func (c *ContextManagerConfig) configPath() (string, error) {
	if viper.IsSet(keyConfig) {
		return viper.GetString(keyConfig), nil
	}

	dir, err := c.defaultConfigDirectory()
	if err != nil {
		return "", err
	}

	return path.Join(dir, c.ConfigName), nil
}

func (c *ContextManagerConfig) defaultConfigDirectory() (string, error) {
	// TODO implement XDG specification?
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(h, "."+c.ConfigDirName), nil
}

func (c *ContextManager) getContexts() (*contextConfig, error) {
	configPath, err := c.cfg.configPath()
	if err != nil {
		return nil, fmt.Errorf(errMsgCannotGetConfigPath, err)
	}

	raw, err := afero.ReadFile(c.cfg.Fs, configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &contextConfig{}, nil
		}

		return nil, fmt.Errorf(errMsgCannotReadConfig, c.cfg.ConfigName, err)
	}

	var ctxs contextConfig
	err = yaml.Unmarshal(raw, &ctxs)

	if ctxCurrent, ok := ctxs.getByName(ctxs.CurrentContext); ok {
		ctxCurrent.IsCurrent = true
	}

	return &ctxs, err
}

func (c *ContextManager) Get(name string) (*Context, error) {
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

func (c *ContextManager) List() ([]*Context, error) {
	ctxs, err := c.getContexts()
	if err != nil {
		return nil, err
	}

	return ctxs.Contexts, nil
}

func (c *ContextManager) Create(rq *Context) (*Context, error) {
	ctxs, err := c.getContexts()
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

	_, _ = fmt.Fprintf(c.cfg.Out, msgContextAdded, successCheck(), color.GreenString(rq.Name))

	return rq, nil
}

func (c *ContextManager) Update(rq *ContextUpdateRequest) (*Context, error) {
	ctxs, err := c.getContexts()
	if err != nil {
		return nil, err
	}

	if rq.Name == "" { // defaults to current context if no name is provided
		if ctxs.CurrentContext == "" {
			return nil, errNoActiveContext
		}
		rq.Name = ctxs.CurrentContext
	}

	ctx, ok := ctxs.getByName(rq.Name)
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

	err = c.writeContextConfig(ctxs)
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

func (c *ContextManager) Delete(name string) (*Context, error) {
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

	err = c.writeContextConfig(ctxs)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(c.cfg.Out, msgContextRemoved, successCheck(), color.GreenString(name))

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
		Activate:       r.IsCurrent,
	}, nil
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
			return fmt.Errorf(errMsgBlankContextField, "Name")
		}
		if context.APIToken == "" {
			return fmt.Errorf(errMsgBlankContextField, "APIToken")
		}
	}

	if len(cs.Contexts) != len(names) {
		return errContextNamesAreUnique
	}

	return nil
}

func (cs *contextConfig) delete(name string) *Context {
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

func (c *ContextManager) GetContextCurrentOrDefault() *Context {
	ctxs, err := c.getContexts()
	if err != nil {
		return defaultCtx()
	}
	ctx, ok := ctxs.getByName(ctxs.CurrentContext)
	if !ok {
		return defaultCtx()
	}
	// TODO deep copy?
	// return ctx.deepCopy()
	return ctx
}

func defaultCtx() *Context {
	return &Context{
		Name:     DefaultContextName,
		APIURL:   pointer.PointerOrNil(viper.GetString(keyAPIURL)),
		APIToken: viper.GetString(keyAPIToken),
	}
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

func DefaultContext(c *ContextManager) (*Context, error) {
	ctxs, err := c.getContexts()
	if err != nil {
		return nil, err
	}

	ctxName := ctxs.CurrentContext
	if viper.IsSet(keyContextName) {
		ctxName = viper.GetString(keyContextName)
	}

	ctx, ok := ctxs.getByName(ctxName)
	if ok {
		return ctx, nil
	}

	defaultCtx := c.GetContextCurrentOrDefault()
	defaultCtx.Name = "default"

	if ctxCurrent, ok := ctxs.getByName(ctxs.CurrentContext); ok {
		ctxCurrent.IsCurrent = false
	}
	defaultCtx.IsCurrent = true
	ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, defaultCtx.Name
	ctxs.Contexts = append(ctxs.Contexts, defaultCtx)

	err = c.writeContextConfig(ctxs)
	if err != nil {
		return nil, fmt.Errorf(errMsgCannotWriteContexts, err)
	}

	ctx = defaultCtx

	return ctx, nil
}

func ContextTable(data any, wide bool) ([]string, [][]string, error) {
	ctxList, ok := data.([]*Context)
	if !ok {
		return nil, nil, errExpectedContextSlice
	}

	if len(ctxList) == 0 {
		return nil, nil, errCreateContextFirst
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
