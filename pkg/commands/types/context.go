package types

import (
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"time"

	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

// Contexts contains all configuration contexts
type Contexts struct {
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

func (cs *Contexts) Get(name string) (*Context, bool) {
	for _, context := range cs.Contexts {
		if context.Name == name {
			return context, true
		}
	}

	return nil, false
}

func (cs *Contexts) List() []*Context {
	return append([]*Context{}, cs.Contexts...)
}

func (cs *Contexts) Validate() error {
	names := map[string]bool{}
	for _, context := range cs.Contexts {
		names[context.Name] = true
	}

	if len(cs.Contexts) != len(names) {
		return fmt.Errorf("context names must be unique")
	}

	return nil
}

func (cs *Contexts) Delete(name string) {
	cs.Contexts = slices.DeleteFunc(cs.Contexts, func(ctx *Context) bool {
		return ctx.Name == name
	})
}

func (c *Config) GetContexts() (*Contexts, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	raw, err := afero.ReadFile(c.Fs, path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Contexts{}, nil
		}

		return nil, fmt.Errorf("unable to read config.yaml: %w", err)
	}

	var ctxs Contexts
	err = yaml.Unmarshal(raw, &ctxs)
	return &ctxs, err
}

func (c *Config) WriteContexts(ctxs *Contexts) error {
	if err := ctxs.Validate(); err != nil {
		return err
	}

	raw, err := yaml.Marshal(ctxs)
	if err != nil {
		return err
	}

	dest, err := ConfigPath()
	if err != nil {
		return err
	}

	// when path is in the default path, we ensure the directory exists
	if defaultPath, err := DefaultConfigDirectory(); err == nil && defaultPath == path.Dir(dest) {
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

func (c *Config) MustDefaultContext() Context {
	ctxs, err := c.GetContexts()
	if err != nil {
		return defaultCtx()
	}

	ctx, ok := ctxs.Get(ctxs.CurrentContext)
	if !ok {
		return defaultCtx()
	}

	return *ctx
}

func defaultCtx() Context {
	return Context{
		ApiURL: pointer.PointerOrNil(viper.GetString("api-url")),
		Token:  viper.GetString("api-token"),
	}
}

func (c *Config) ContextListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctxs, err := c.GetContexts()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var names []string
	for _, ctx := range ctxs.Contexts {
		names = append(names, ctx.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
