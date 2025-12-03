package genericcli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// contextManagerTestCase defines a test case for ContextManager operations
type contextManagerTestCase[T any] struct {
	Name          string
	ConfigContent *contextConfig
	Setup         func(t *testing.T, manager *ContextManager) error
	Run           func(t *testing.T, manager *ContextManager) (T, error)
	wantErr       error
	want          T
}

func ctxMinimal() *Context {
	return &Context{
		Name:     "ctx1",
		APIToken: "token1",
	}
}

func ctxWithProvider() *Context {
	return &Context{
		Name:     "ctx2",
		APIToken: "token2",
		Provider: "foo",
	}
}

func ctxFull() *Context {
	return &Context{
		Name:           "ctx3",
		APIURL:         pointer.Pointer("http://foo.bar"),
		APIToken:       "token3",
		DefaultProject: "project3",
		Timeout:        pointer.Pointer(time.Duration(100)),
		Provider:       "foo",
	}
}

func ctxNew() *Context {
	return &Context{
		Name:     "ctxNew",
		APIToken: "tokenNew",
	}
}

func ctxList() []*Context {
	return []*Context{
		ctxMinimal(),
		ctxWithProvider(),
		ctxFull(),
	}
}

func contextConfigWithActiveUnsetCurrentUnset() *contextConfig {
	return &contextConfig{
		CurrentContext:  "",
		PreviousContext: "",
		Contexts:        ctxList(),
	}
}

func contextConfigWithActiveSetCurrentUnset() *contextConfig {
	list := ctxList()
	return &contextConfig{
		CurrentContext:  list[0].Name,
		PreviousContext: list[1].Name,
		Contexts:        list,
	}
}

func contextConfigWithActiveSetCurrentSet() *contextConfig {
	list := ctxList()
	markAsCurrent(list[0])
	return &contextConfig{
		CurrentContext:  list[0].Name,
		PreviousContext: list[1].Name,
		Contexts:        list,
	}
}

func markAsCurrent(ctx *Context) *Context {
	ctx.IsCurrent = true
	return ctx
}

func setupFs(t *testing.T) (afero.Fs, string) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := path.Join(homeDir, ".test-config")

	require.NoError(t, fs.MkdirAll(configDir, 0755))
	return fs, configDir
}

func newTestManager(t *testing.T) *ContextManager {
	fs, configDir := setupFs(t)
	return NewContextManager(&ContextCmdConfig{
		BinaryName:      os.Args[0],
		ConfigDirName:   configDir,
		ConfigName:      "config.yaml",
		Fs:              fs,
		Out:             io.Discard,
		ListPrinter:     func() printers.Printer { return printers.NewYAMLPrinter() },
		DescribePrinter: func() printers.Printer { return printers.NewYAMLPrinter() },
	})
}

func runManagerTests[T any](t *testing.T, tests []contextManagerTestCase[T]) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			manager := newTestManager(t)

			if test.ConfigContent == nil {
				test.ConfigContent = &contextConfig{
					Contexts: []*Context{},
				}
			}
			require.NoError(t, manager.writeContextConfig(test.ConfigContent))

			if test.Setup != nil {
				err := test.Setup(t, manager)
				require.NoError(t, err)
			}

			got, err := test.Run(t, manager)

			if diff := cmp.Diff(test.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Diff = %s", diff)
			}
		})
	}
}

func TestContextManager_Get(t *testing.T) {
	tests := []contextManagerTestCase[*Context]{
		{
			Name:          "get existing context",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       nil,
			want:          ctxMinimal(),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get(ctxMinimal().Name)
			},
		},
		{
			Name:          "get non-existent context",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       fmt.Errorf("context \"%s\" not found", "nonexistent"),
			want:          nil,
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get("nonexistent")
			},
		},
		{
			Name:          "get from empty file",
			ConfigContent: nil,
			wantErr:       fmt.Errorf("context \"%s\" not found", "nonexistent"),
			want:          nil,
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get("nonexistent")
			},
		},
		{
			Name:          "get active context",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want:          markAsCurrent(ctxMinimal()),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get(ctxMinimal().Name)
			},
		},
	}

	runManagerTests(t, tests)
}

func TestContextManager_GetCurrentContext(t *testing.T) {
	tests := []contextManagerTestCase[*Context]{
		{
			Name:          "current is set",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want:          markAsCurrent(ctxMinimal()),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetCurrentContext()
			},
		},
		{
			Name: "current is non-existent",
			ConfigContent: func() *contextConfig {
				c := contextConfigWithActiveUnsetCurrentUnset()
				c.CurrentContext = "nonexistent"
				return c
			}(),
			wantErr: nil,
			want:    nil,
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetCurrentContext()
			},
		},
		{
			Name:          "current is not set",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       nil,
			want:          nil,
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetCurrentContext()
			},
		},
		{
			Name:          "empty file",
			ConfigContent: nil,
			wantErr:       nil,
			want:          nil,
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetCurrentContext()
			},
		},
	}

	runManagerTests(t, tests)
}
func TestContextManager_GetContextCurrentOrDefault(t *testing.T) {
	tests := []contextManagerTestCase[*Context]{
		{
			Name:          "current is set",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want:          markAsCurrent(ctxMinimal()),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetContextCurrentOrDefault(), nil
			},
		},
		{
			Name:          "current is not set",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       nil,
			want:          defaultCtx(),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetContextCurrentOrDefault(), nil
			},
		},
		{
			Name: "current is to non-existent context",
			ConfigContent: func() *contextConfig {
				c := contextConfigWithActiveUnsetCurrentUnset()
				c.CurrentContext = "nonexistent"
				return c
			}(),
			wantErr: nil,
			want:    defaultCtx(),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetContextCurrentOrDefault(), nil
			},
		},
	}

	runManagerTests(t, tests)
}

func TestContextManager_List(t *testing.T) {
	tests := []contextManagerTestCase[[]*Context]{
		{
			Name:          "no active contexts",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       nil,
			want:          contextConfigWithActiveUnsetCurrentUnset().Contexts,
			Run: func(t *testing.T, manager *ContextManager) ([]*Context, error) {
				return manager.List()
			},
		},
		{
			Name:          "active context is present",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want:          contextConfigWithActiveSetCurrentSet().Contexts,
			Run: func(t *testing.T, manager *ContextManager) ([]*Context, error) {
				return manager.List()
			},
		},
		{
			Name:          "list from empty file",
			ConfigContent: nil,
			wantErr:       nil,
			want:          []*Context{},
			Run: func(t *testing.T, manager *ContextManager) ([]*Context, error) {
				return manager.List()
			},
		},
	}

	runManagerTests(t, tests)
}

func TestContextManager_Create(t *testing.T) {
	createHelperFunc := func(ctx *Context) func(*testing.T, *ContextManager) (*contextConfig, error) {
		return func(t *testing.T, manager *ContextManager) (*contextConfig, error) {
			_, err := manager.Create(ctx)
			if err != nil {
				return nil, err
			}
			return manager.getContextConfig()
		}
	}

	tests := []contextManagerTestCase[*contextConfig]{
		{
			Name:          "first context auto-activates",
			ConfigContent: nil,
			wantErr:       nil,
			want: &contextConfig{
				CurrentContext:  ctxMinimal().Name,
				PreviousContext: "",
				Contexts:        []*Context{markAsCurrent(ctxMinimal())},
			},
			Run: createHelperFunc(ctxMinimal()),
		},
		{
			Name:          "create context with activate flag",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want: func() *contextConfig {
				ctxs := contextConfigWithActiveSetCurrentUnset()
				new := markAsCurrent(ctxNew())

				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, new.Name
				ctxs.Contexts = append(ctxs.Contexts, new)

				return ctxs
			}(),
			Run: createHelperFunc(markAsCurrent(ctxNew())),
		},
		{
			Name:          "create duplicate context Name fails",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       errors.New("context names must be unique"),
			want:          nil,
			Run: createHelperFunc(&Context{
				Name:     ctxMinimal().Name,
				APIToken: "token123",
			}),
		},
		{
			Name:          "create context without token fails",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       fmt.Errorf("context field \"%s\" cannot be blank", "APIToken"),
			want:          nil,
			Run: createHelperFunc(&Context{
				Name:     "notoken",
				APIToken: "",
			}),
		},
	}

	runManagerTests(t, tests)
}

func TestContextManager_Update(t *testing.T) {
	updateHelperFunc := func(rq *ContextUpdateRequest) func(*testing.T, *ContextManager) (*contextConfig, error) {
		return func(t *testing.T, manager *ContextManager) (*contextConfig, error) {
			_, err := manager.Update(rq)
			if err != nil {
				return nil, err
			}
			return manager.getContextConfig()
		}
	}

	tests := []contextManagerTestCase[*contextConfig]{
		{
			Name:          "update existing context (all fields)",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       nil,
			want: func() *contextConfig {
				want := contextConfigWithActiveUnsetCurrentUnset()
				want.Contexts[0].APIURL = pointer.Pointer("newAPIURL")
				want.Contexts[0].APIToken = "newAPIToken"
				want.Contexts[0].DefaultProject = "newProject"
				want.Contexts[0].Timeout = pointer.Pointer(time.Duration(100))
				want.Contexts[0].Provider = "newProvider"
				return want
			}(),
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:           ctxMinimal().Name,
				APIURL:         pointer.Pointer("newAPIURL"),
				APIToken:       pointer.Pointer("newAPIToken"),
				DefaultProject: pointer.Pointer("newProject"),
				Timeout:        pointer.Pointer(time.Duration(100)),
				Provider:       pointer.Pointer("newProvider"),
			}),
		},
		{
			Name:          "update with activate flag",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want: func() *contextConfig {
				ctxs := contextConfigWithActiveSetCurrentUnset()
				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, ctxFull().Name
				ctxs.Contexts[2].IsCurrent = true
				return ctxs
			}(),
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:      ctxFull().Name,
				IsCurrent: true,
			}),
		},
		{
			Name:          "update non-existent context",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       fmt.Errorf("context \"%s\" not found", "nonexistent"),
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:           "nonexistent",
				DefaultProject: pointer.Pointer("foo"),
			}),
		},
		{
			Name:          "update current context without Name",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want: func() *contextConfig {
				ctxs := contextConfigWithActiveSetCurrentSet()
				ctxs.Contexts[0].Provider = "foo"
				return ctxs
			}(),
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:     "",
				Provider: pointer.Pointer("foo"),
			}),
		},
		{
			Name:          "fail with no current and no Name",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       errors.New("no context currently active"),
			want:          nil,
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:     "",
				Provider: pointer.Pointer("foo"),
			}),
		},
	}

	runManagerTests(t, tests)
}

func TestContextManager_Delete(t *testing.T) {
	deleteHelperFunc := func(name string) func(*testing.T, *ContextManager) (*contextConfig, error) {
		return func(t *testing.T, manager *ContextManager) (*contextConfig, error) {
			_, err := manager.Delete(name)
			if err != nil {
				return nil, err
			}
			return manager.getContextConfig()
		}
	}

	tests := []contextManagerTestCase[*contextConfig]{
		{
			Name:          "delete existing context",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       nil,
			want: &contextConfig{
				Contexts: []*Context{ctxMinimal(), ctxWithProvider()},
			},
			Run: deleteHelperFunc(ctxFull().Name),
		},
		{
			Name:          "delete active context",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want: func() *contextConfig {
				ctxs := contextConfigWithActiveSetCurrentUnset()
				ctxs.Contexts = ctxs.Contexts[1:]
				ctxs.CurrentContext = ""
				return ctxs
			}(),
			Run: deleteHelperFunc(ctxMinimal().Name),
		},
		{
			Name:          "delete previous context",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want: &contextConfig{
				CurrentContext: ctxMinimal().Name,
				Contexts:       []*Context{markAsCurrent(ctxMinimal()), ctxFull()},
			},
			Run: deleteHelperFunc(ctxWithProvider().Name),
		},
		{
			Name:          "delete non-existent context",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       fmt.Errorf("context \"%s\" not found", "nonexistent"),
			want:          nil,
			Run:           deleteHelperFunc("nonexistent"),
		},
	}

	runManagerTests(t, tests)
}

func TestContexts_Validate(t *testing.T) {
	tests := []struct {
		Name    string
		ctxs    *contextConfig
		wantErr error
	}{
		{
			Name:    "valid contexts",
			ctxs:    contextConfigWithActiveUnsetCurrentUnset(),
			wantErr: nil,
		},
		{
			Name: "duplicate Names",
			ctxs: &contextConfig{
				Contexts: []*Context{
					{Name: "ctx1", APIToken: "token1"},
					{Name: "ctx1", APIToken: "token2"},
				},
			},
			wantErr: errors.New("context names must be unique"),
		},
		{
			Name: "blank Name",
			ctxs: &contextConfig{
				Contexts: []*Context{
					{Name: "", APIToken: "token1"},
				},
			},
			wantErr: fmt.Errorf("context field \"%s\" cannot be blank", "Name"),
		},
		{
			Name: "blank token",
			ctxs: &contextConfig{
				Contexts: []*Context{
					{Name: "ctx1", APIToken: ""},
				},
			},
			wantErr: fmt.Errorf("context field \"%s\" cannot be blank", "APIToken"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			err := tt.ctxs.validate()
			if tt.wantErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestContextManager_writeContextConfig(t *testing.T) {
	tests := []struct {
		Name          string
		InputContexts *contextConfig
		WantErr       error
		ValidateFile  func(t *testing.T, fs afero.Fs, configPath string)
	}{
		{
			Name:          "write valid contexts",
			InputContexts: contextConfigWithActiveSetCurrentUnset(),
			WantErr:       nil,
			ValidateFile: func(t *testing.T, fs afero.Fs, configPath string) {
				exists, err := afero.Exists(fs, configPath)
				require.NoError(t, err)
				require.True(t, exists, "config file should exist")

				content, err := afero.ReadFile(fs, configPath)
				require.NoError(t, err)

				var ctxs contextConfig
				err = yaml.Unmarshal(content, &ctxs)
				require.NoError(t, err)
				require.Equal(t, ctxMinimal().Name, ctxs.CurrentContext)
				require.Equal(t, ctxWithProvider().Name, ctxs.PreviousContext)
				require.Len(t, ctxs.Contexts, 3)
			},
		},
		{
			Name:          "write empty contexts",
			InputContexts: &contextConfig{},
			WantErr:       nil,
			ValidateFile: func(t *testing.T, fs afero.Fs, configPath string) {
				content, err := afero.ReadFile(fs, configPath)
				require.NoError(t, err)

				var ctxs contextConfig
				err = yaml.Unmarshal(content, &ctxs)
				require.NoError(t, err)
				require.Empty(t, ctxs.CurrentContext)
				require.Empty(t, ctxs.Contexts)
			},
		},
		{
			Name: "fail on duplicate context names",
			InputContexts: &contextConfig{
				Contexts: []*Context{
					{Name: "duplicate", APIToken: "token1"},
					{Name: "duplicate", APIToken: "token2"},
				},
			},
			WantErr: errors.New("context names must be unique"),
		},
		{
			Name: "fail on blank context name",
			InputContexts: &contextConfig{
				Contexts: []*Context{
					{Name: "", APIToken: "token1"},
				},
			},
			WantErr: fmt.Errorf("context field \"%s\" cannot be blank", "Name"),
		},
		{
			Name: "fail on blank API token",
			InputContexts: &contextConfig{
				Contexts: []*Context{
					{Name: "ctx1", APIToken: ""},
				},
			},
			WantErr: fmt.Errorf("context field \"%s\" cannot be blank", "APIToken"),
		},
		{
			Name:          "create config directory if in default path",
			InputContexts: contextConfigWithActiveUnsetCurrentUnset(),
			WantErr:       nil,
			ValidateFile: func(t *testing.T, fs afero.Fs, configPath string) {
				// Verify directory was created
				dirExists, err := afero.DirExists(fs, path.Dir(configPath))
				require.NoError(t, err)
				require.True(t, dirExists, "config directory should be created")

				// Verify file permissions (0600)
				info, err := fs.Stat(configPath)
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0600), info.Mode().Perm())
			},
		},
		{
			Name: "preserve context ordering",
			InputContexts: &contextConfig{
				CurrentContext: "ctx2",
				Contexts: []*Context{
					ctxFull(),
					ctxMinimal(),
					ctxWithProvider(),
				},
			},
			WantErr: nil,
			ValidateFile: func(t *testing.T, fs afero.Fs, configPath string) {
				content, err := afero.ReadFile(fs, configPath)
				require.NoError(t, err)

				var ctxs contextConfig
				err = yaml.Unmarshal(content, &ctxs)
				require.NoError(t, err)
				require.Len(t, ctxs.Contexts, 3)
				require.Equal(t, "ctx3", ctxs.Contexts[0].Name)
				require.Equal(t, "ctx1", ctxs.Contexts[1].Name)
				require.Equal(t, "ctx2", ctxs.Contexts[2].Name)
			},
		},
		{
			Name: "write contexts with all fields populated",
			InputContexts: &contextConfig{
				CurrentContext: ctxFull().Name,
				Contexts: []*Context{
					ctxFull(),
				},
			},
			WantErr: nil,
			ValidateFile: func(t *testing.T, fs afero.Fs, configPath string) {
				content, err := afero.ReadFile(fs, configPath)
				require.NoError(t, err)

				var ctxs contextConfig
				err = yaml.Unmarshal(content, &ctxs)
				require.NoError(t, err)
				require.Len(t, ctxs.Contexts, 1)
				require.NotNil(t, ctxs.Contexts[0].APIURL)
				require.Equal(t, "http://foo.bar", *ctxs.Contexts[0].APIURL)
				require.Equal(t, "foo", ctxs.Contexts[0].Provider)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			manager := newTestManager(t)
			configPath, err := manager.cfg.configPath()
			require.NoError(t, err)

			err = manager.writeContextConfig(tt.InputContexts)

			if tt.WantErr != nil {
				require.Error(t, err)
				if diff := cmp.Diff(tt.WantErr, err, testcommon.ErrorStringComparer()); diff != "" {
					t.Errorf("error diff (+got -want):\n %s", diff)
				}
				return
			}

			require.NoError(t, err)
			if tt.ValidateFile != nil {
				tt.ValidateFile(t, manager.cfg.Fs, configPath)
			}
		})
	}
}

func TestContextManager_getContextsConfig(t *testing.T) {
	tests := []contextManagerTestCase[*contextConfig]{
		{
			Name:          "read existing contexts && IsCurrent is set",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			wantErr:       nil,
			want:          contextConfigWithActiveSetCurrentSet(),
			Run: func(t *testing.T, manager *ContextManager) (*contextConfig, error) {
				return manager.getContextConfig()
			},
		},
		{
			Name:          "read empty config returns empty contexts",
			ConfigContent: nil,
			wantErr:       nil,
			want: &contextConfig{
				Contexts: []*Context{},
			},
			Run: func(t *testing.T, manager *ContextManager) (*contextConfig, error) {
				return manager.getContextConfig()
			},
		},
		{
			Name:          "read contexts without active context",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			wantErr:       nil,
			want:          contextConfigWithActiveUnsetCurrentUnset(),
			Run: func(t *testing.T, manager *ContextManager) (*contextConfig, error) {
				return manager.getContextConfig()
			},
		},
		{
			Name: "preserve all context fields",
			ConfigContent: &contextConfig{
				CurrentContext: ctxFull().Name,
				Contexts: []*Context{
					ctxFull(),
				},
			},
			wantErr: nil,
			want: &contextConfig{
				CurrentContext: ctxFull().Name,
				Contexts:       []*Context{markAsCurrent(ctxFull())},
			},
			Run: func(t *testing.T, manager *ContextManager) (*contextConfig, error) {
				return manager.getContextConfig()
			},
		},
		{
			Name: "handle current context not in list",
			ConfigContent: &contextConfig{
				CurrentContext: "nonexistent",
				Contexts:       ctxList(),
			},
			wantErr: nil,
			want: &contextConfig{
				CurrentContext: "nonexistent",
				Contexts:       ctxList(),
			},
			Run: func(t *testing.T, manager *ContextManager) (*contextConfig, error) {
				return manager.getContextConfig()
			},
		},
	}

	runManagerTests(t, tests)
}

func TestContextManager_writeContexts_getContextsConfig_RoundTrip(t *testing.T) {
	helperFunc := func(ctxs *contextConfig) func(*testing.T, *ContextManager) (*contextConfig, error) {
		return func(t *testing.T, manager *ContextManager) (*contextConfig, error) {
			err := manager.writeContextConfig(ctxs)
			if err != nil {
				return nil, err
			}
			return manager.getContextConfig()
		}
	}
	tests := []contextManagerTestCase[*contextConfig]{
		{
			Name:          "round trip with active context",
			ConfigContent: nil,
			wantErr:       nil,
			want:          contextConfigWithActiveSetCurrentSet(),
			Run:           helperFunc(contextConfigWithActiveSetCurrentUnset()),
		},
		{
			Name:          "round trip without active context",
			ConfigContent: nil,
			wantErr:       nil,
			want:          contextConfigWithActiveUnsetCurrentUnset(),
			Run:           helperFunc(contextConfigWithActiveUnsetCurrentUnset()),
		},
		{
			Name:          "round trip with all optional fields",
			ConfigContent: nil,
			wantErr:       nil,
			want: &contextConfig{
				CurrentContext:  ctxFull().Name,
				PreviousContext: ctxMinimal().Name,
				Contexts:        []*Context{markAsCurrent(ctxFull()), ctxMinimal()},
			},
			Run: helperFunc(&contextConfig{
				CurrentContext:  ctxFull().Name,
				PreviousContext: ctxMinimal().Name,
				Contexts:        []*Context{ctxFull(), ctxMinimal()},
			}),
		},
	}

	runManagerTests(t, tests)
}

// Console tests below

type consoleTestCase[T any] struct {
	Name          string
	ConfigContent *contextConfig
	Args          []string
	Setup         func(t *testing.T, cmd *cobra.Command) error
	wantErr       error
	wantOut       string
	want          T
}

func runConsoleTests[T any](t *testing.T, tests []consoleTestCase[T]) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			fs, configDir := setupFs(t)
			mgr := NewContextManager(&ContextCmdConfig{
				BinaryName:      os.Args[0],
				ConfigDirName:   configDir,
				ConfigName:      "config.yaml",
				Fs:              fs,
				Out:             io.Discard,
				ListPrinter:     func() printers.Printer { return printers.NewYAMLPrinter() },
				DescribePrinter: func() printers.Printer { return printers.NewYAMLPrinter() },
			})

			if test.ConfigContent == nil {
				test.ConfigContent = &contextConfig{}
			}
			require.NoError(t, mgr.writeContextConfig(test.ConfigContent))

			buf := &bytes.Buffer{}
			cmd := getNewContextCmd(fs, io.Writer(buf), configDir)

			cmd.SetArgs(test.Args)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			if test.Setup != nil {
				err := test.Setup(t, cmd)
				require.NoError(t, err)
			}

			err := cmd.Execute()

			if diff := cmp.Diff(test.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
				return
			}
			if diff := cmp.Diff(test.wantOut, buf.String()); diff != "" {
				t.Errorf("Diff = %s", diff)
			}

			result, err := mgr.getContextConfig()
			require.NoError(t, err)
			if diff := cmp.Diff(test.want, result); diff != "" {
				t.Errorf("Diff = %s", diff)
			}
		})
	}
}

func newPrinterFromCLI(c *ContextCmdConfig) printers.Printer {
	var printer printers.Printer

	switch format := viper.GetString("output-format"); format {
	case "yaml":
		printer = printers.NewProtoYAMLPrinter().WithFallback(true).WithOut(c.Out)
	case "json":
		printer = printers.NewProtoJSONPrinter().WithFallback(true).WithOut(c.Out)
	case "yamlraw":
		printer = printers.NewYAMLPrinter().WithOut(c.Out)
	case "jsonraw":
		printer = printers.NewJSONPrinter().WithOut(c.Out)
	case "template":
		printer = printers.NewTemplatePrinter(viper.GetString("template")).WithOut(c.Out)
	case "table", "wide", "markdown":
		fallthrough
	default:
		cfg := &printers.TablePrinterConfig{
			ToHeaderAndRows: ContextTable,
			Wide:            format == "wide",
			Markdown:        format == "markdown",
			NoHeaders:       viper.GetBool("no-headers"),
			Out:             c.Out,
		}
		tablePrinter := printers.NewTablePrinter(cfg).WithOut(c.Out)
		printer = tablePrinter
	}

	if viper.IsSet("force-color") {
		enabled := viper.GetBool("force-color")
		if enabled {
			color.NoColor = false
		} else {
			color.NoColor = true
		}
	}

	return printer
}

func getNewContextCmd(fs afero.Fs, buf io.Writer, configDir string) *cobra.Command {
	c := &ContextCmdConfig{
		ConfigDirName:         configDir,
		BinaryName:            os.Args[0],
		Fs:                    fs,
		In:                    nil,
		Out:                   buf,
		ProjectListCompletion: nil,
	}

	tablePrinter := newPrinterFromCLI(c)
	c.ListPrinter = func() printers.Printer { return tablePrinter }
	c.DescribePrinter = func() printers.Printer { return tablePrinter }

	return NewContextCmd(c)
}

func TestContextManager_SwitchContext(t *testing.T) {
	tests := []consoleTestCase[*contextConfig]{
		{
			Name:          "switch to existing context",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			Args:          []string{ctxFull().Name},
			wantErr:       nil,
			want: func() *contextConfig {
				ctxs := contextConfigWithActiveSetCurrentUnset()
				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, ctxFull().Name
				ctxs.Contexts[2].IsCurrent = true
				return ctxs
			}(),
			wantOut: fmt.Sprintf("✔ Switched context to \"%s\"\n", ctxFull().Name),
		},
		{
			Name:          "switch to the same context",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			Args:          []string{ctxMinimal().Name},
			wantErr:       nil,
			want:          contextConfigWithActiveSetCurrentSet(),
			wantOut:       fmt.Sprintf("✔ Context \"%s\" is already active\n", ctxMinimal().Name),
		},
		{
			Name:          "switch to previous context using dash",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			Args:          []string{"-"},
			wantErr:       nil,
			want: func() *contextConfig {
				ctxs := contextConfigWithActiveSetCurrentUnset()
				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, ctxs.PreviousContext
				ctxs.Contexts[1].IsCurrent = true
				return ctxs
			}(),
			wantOut: fmt.Sprintf("✔ Switched context to \"%s\"\n", ctxWithProvider().Name),
		},
		{
			Name:          "switch to previous when none exists",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			Args:          []string{"-"},
			wantErr:       errors.New("no previous context found"),
			want:          contextConfigWithActiveUnsetCurrentUnset(),
		},
		{
			Name:          "switch to non-existent context",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			Args:          []string{"nonexistent"},
			wantErr:       fmt.Errorf("context \"%s\" not found", "nonexistent"),
			want:          contextConfigWithActiveSetCurrentSet(),
		},
		{
			Name: "switch to previous when no context is active",
			ConfigContent: func() *contextConfig {
				ctxs := contextConfigWithActiveSetCurrentUnset()
				ctxs.CurrentContext = ""
				return ctxs
			}(),
			Args:    []string{"-"},
			wantErr: nil,
			want: func() *contextConfig {
				ctxs := contextConfigWithActiveSetCurrentUnset()
				ctxs.PreviousContext, ctxs.CurrentContext = "", ctxWithProvider().Name
				ctxs.Contexts[1].IsCurrent = true
				return ctxs
			}(),
			wantOut: fmt.Sprintf("✔ Switched context to \"%s\"\n", ctxWithProvider().Name),
		},
	}

	runConsoleTests(t, tests)
}

func TestContextManager_SetProject(t *testing.T) {
	tests := []consoleTestCase[*contextConfig]{
		{
			Name:          "set project on active context",
			ConfigContent: contextConfigWithActiveSetCurrentUnset(),
			Args:          []string{"set-project", "new-project"},
			wantErr:       nil,
			wantOut:       fmt.Sprintf("✔ Updated context \"%s\"\n✔ Switched context default project to \"new-project\"\n", ctxMinimal().Name),
			want: func() *contextConfig {
				ctxs := contextConfigWithActiveSetCurrentSet()
				ctxs.Contexts[0].DefaultProject = "new-project"
				return ctxs
			}(),
		},
		{
			Name:          "set project with no active context",
			ConfigContent: contextConfigWithActiveUnsetCurrentUnset(),
			Args:          []string{"set-project", "new-project"},
			wantErr:       errors.New("no context currently active"),
			want:          contextConfigWithActiveUnsetCurrentUnset(),
		},
	}

	runConsoleTests(t, tests)
}
