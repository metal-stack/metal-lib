package genericcli

import (
	"fmt"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ManagerTestCase defines a test case for ContextManager operations
type ManagerTestCase[T any] struct {
	Name        string
	FileContent *contexts
	Setup       func(t *testing.T, manager *ContextManager) error
	Run         func(t *testing.T, manager *ContextManager) (T, error)
	wantErr     error
	want        T
}

func ctx1() *Context {
	return &Context{
		Name:     "ctx1",
		APIToken: "token1",
	}
}

func ctx2() *Context {
	return &Context{
		Name:     "ctx2",
		APIToken: "token2",
		Provider: "foo",
	}
}

func ctx3() *Context {
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
		ctx1(),
		ctx2(),
		ctx3(),
	}
}

func contextsActiveUnsetCurrentUnset() *contexts {
	return &contexts{
		CurrentContext:  "",
		PreviousContext: "",
		Contexts:        ctxList(),
	}
}

func contextsActiveSetCurrentUnset() *contexts {
	list := ctxList()
	return &contexts{
		CurrentContext:  list[0].Name,
		PreviousContext: list[1].Name,
		Contexts:        list,
	}
}

func contextsActiveSetCurrentSet() *contexts {
	list := ctxList()
	current(list[0])
	return &contexts{
		CurrentContext:  list[0].Name,
		PreviousContext: list[1].Name,
		Contexts:        list,
	}
}

func current(ctx *Context) *Context {
	ctx.IsCurrent = true
	return ctx
}

func setupFs(t *testing.T) (afero.Fs, string) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := path.Join(homeDir, ".test-config")

	// Mock home directory
	// os.Setenv("HOME", homeDir)

	// Create config directory
	require.NoError(t, fs.MkdirAll(configDir, 0755))
	return fs, configDir
}

func newTestManager(t *testing.T) *ContextManager {
	fs, configDir := setupFs(t)
	return NewContextManager(&ContextConfig{
		BinaryName:      os.Args[0],
		ConfigDirName:   configDir,
		ConfigName:      "config.yaml",
		Fs:              fs,
		Out:             io.Discard,
		ListPrinter:     func() printers.Printer { return printers.NewYAMLPrinter() },
		DescribePrinter: func() printers.Printer { return printers.NewYAMLPrinter() },
	})
}

func managerTest[T any](t *testing.T, tests []ManagerTestCase[T]) {
	for _, test := range tests {
		managerTestOne(t, test)
	}
}

func managerTestOne[T any](t *testing.T, tt ManagerTestCase[T]) {
	t.Run(tt.Name, func(t *testing.T) {
		manager := newTestManager(t)

		if tt.FileContent == nil {
			tt.FileContent = &contexts{}
		}
		require.NoError(t, manager.writeContexts(tt.FileContent))

		if tt.Setup != nil {
			err := tt.Setup(t, manager)
			require.NoError(t, err)
		}

		got, err := tt.Run(t, manager)

		if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
			t.Errorf("error diff (+got -want):\n %s", diff)
			return
		}
		if diff := cmp.Diff(tt.want, got); diff != "" {
			t.Errorf("Diff = %s", diff)
		}
	})
}

func TestContextManager_Get(t *testing.T) {
	tests := []ManagerTestCase[*Context]{
		{
			Name:        "get existing context",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     nil,
			want:        ctx1(),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get(ctx1().Name)
			},
		},
		{
			Name:        "get non-existent context",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     fmt.Errorf(errMsgContextNotFound, "nonexistent"),
			want:        nil,
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get("nonexistent")
			},
		},
		{
			Name:        "get from empty file",
			FileContent: nil,
			wantErr:     fmt.Errorf(errMsgContextNotFound, "nonexistent"),
			want:        nil,
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get("nonexistent")
			},
		},
		{
			Name:        "get active context",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want:        current(ctx1()),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get(ctx1().Name)
			},
		},
	}

	managerTest(t, tests)
}

func TestContextManager_GetCurrentContext(t *testing.T) {
	tests := []ManagerTestCase[*Context]{
		{
			Name:        "current is set",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want:        current(ctx1()),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetCurrentContext()
			},
		},
		{
			Name:        "current is not set",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     nil,
			want:        nil,
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetCurrentContext()
			},
		},
		{
			Name:        "empty file",
			FileContent: nil,
			wantErr:     nil,
			want:        nil,
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetCurrentContext()
			},
		},
	}

	managerTest(t, tests)
}
func TestContextManager_GetContextCurrentOrDefault(t *testing.T) {
	tests := []ManagerTestCase[*Context]{
		{
			Name:        "current is set",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want:        current(ctx1()),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetContextCurrentOrDefault(), nil
			},
		},
		{
			Name:        "current is not set",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     nil,
			want:        defaultCtx(),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.GetContextCurrentOrDefault(), nil
			},
		},
	}

	managerTest(t, tests)
}

func TestContextManager_DefaultContext(t *testing.T) {
	type defaultContextResult struct {
		ReturnedCtx  *Context  // The *Context returned by the function
		FileContents *contexts // The state in the config file *after* the run
	}

	runFunc := func(t *testing.T, manager *ContextManager) (defaultContextResult, error) {
		returnedCtx, err := DefaultContext(manager)
		if err != nil {
			return defaultContextResult{}, err
		}

		ctxs, err := manager.getContexts()
		return defaultContextResult{
			ReturnedCtx:  returnedCtx,
			FileContents: ctxs,
		}, err
	}

	tests := []ManagerTestCase[defaultContextResult]{
		{
			Name:        "viper override finds existing context (no switching)",
			FileContent: contextsActiveSetCurrentUnset(), // "ctx1" is current
			wantErr:     nil,
			want: defaultContextResult{
				ReturnedCtx:  ctx2(),
				FileContents: contextsActiveSetCurrentSet(),
			},
			Setup: func(t *testing.T, manager *ContextManager) error {
				viper.Reset()
				t.Cleanup(viper.Reset)
				viper.Set(keyContextName, ctx2().Name) // Override to "ctx2"
				return nil
			},
			Run: runFunc,
		},
		{
			Name:        "no viper override, current context is found (no switching)",
			FileContent: contextsActiveSetCurrentUnset(), // "ctx1" is current
			wantErr:     nil,
			want: defaultContextResult{
				ReturnedCtx:  current(ctx1()),
				FileContents: contextsActiveSetCurrentSet(),
			},
			Setup: func(t *testing.T, manager *ContextManager) error {
				viper.Reset() // Ensure viper.IsSet is false
				return nil
			},
			Run: runFunc,
		},
		{
			Name:        "viper override, context not found, creates default (with switching)",
			FileContent: contextsActiveSetCurrentUnset(), // current="ctx1", prev="ctx2"
			wantErr:     nil,
			want: func() defaultContextResult {
				want := current(ctx1())
				want.Name = "default"
				return defaultContextResult{
					ReturnedCtx: want, // TODO WARNING default is current with changed name. Do we want this?
					FileContents: func() *contexts {
						return &contexts{
							CurrentContext:  want.Name,
							PreviousContext: ctx1().Name,
							Contexts:        append(ctxList(), want),
						}
					}(),
				}
			}(),
			Setup: func(t *testing.T, manager *ContextManager) error {
				viper.Reset()
				t.Cleanup(viper.Reset)
				viper.Set(keyContextName, "nonexistent")
				return nil
			},
			Run: runFunc,
		},
		{
			Name:        "no viper override, current context not set, fails",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     fmt.Errorf(errMsgCannotWriteContexts, fmt.Errorf(errMsgBlankContextField, "APIToken")),
			want:        defaultContextResult{},
			Setup: func(t *testing.T, manager *ContextManager) error {
				viper.Reset()
				return nil
			},
			Run: runFunc,
		},
		{
			Name:        "no viper override, current context not set, creates default with APIToken from viper (with switching)",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     nil,
			want: func() defaultContextResult {
				want := defaultCtx()
				want.APIToken = "tokenDefault"
				want.IsCurrent = true
				return defaultContextResult{
					ReturnedCtx: want,
					FileContents: func() *contexts {
						return &contexts{
							CurrentContext:  "default",
							PreviousContext: "",
							Contexts:        append(ctxList(), want),
						}
					}(),
				}
			}(),
			Setup: func(t *testing.T, manager *ContextManager) error {
				viper.Reset()
				t.Cleanup(viper.Reset)
				viper.Set(keyAPIToken, "tokenDefault")
				return nil
			},
			Run: runFunc,
		},
	}

	managerTest(t, tests)
}

func TestContextManager_List(t *testing.T) {
	tests := []ManagerTestCase[[]*Context]{
		{
			Name:        "no active contexts",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     nil,
			want:        contextsActiveUnsetCurrentUnset().Contexts,
			Run: func(t *testing.T, manager *ContextManager) ([]*Context, error) {
				return manager.List()
			},
		},
		{
			Name:        "active context is present",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want:        contextsActiveSetCurrentSet().Contexts,
			Run: func(t *testing.T, manager *ContextManager) ([]*Context, error) {
				return manager.List()
			},
		},
		{
			Name:        "list from empty file",
			FileContent: nil,
			wantErr:     nil,
			want:        []*Context{},
			Run: func(t *testing.T, manager *ContextManager) ([]*Context, error) {
				return manager.List()
			},
		},
	}

	managerTest(t, tests)
}

func TestContextManager_Create(t *testing.T) {
	createHelperFunc := func(ctx *Context) func(*testing.T, *ContextManager) (*contexts, error) {
		return func(t *testing.T, manager *ContextManager) (*contexts, error) {
			_, err := manager.Create(ctx)
			if err != nil {
				return nil, err
			}
			return manager.getContexts()
		}
	}

	tests := []ManagerTestCase[*contexts]{
		{
			Name:        "first context auto-activates",
			FileContent: nil,
			wantErr:     nil,
			want: &contexts{
				CurrentContext:  ctx1().Name,
				PreviousContext: "",
				Contexts:        []*Context{current(ctx1())},
			},
			Run: createHelperFunc(ctx1()),
		},
		{
			Name:        "create context with activate flag",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsActiveSetCurrentUnset()
				new := current(ctxNew())

				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, new.Name
				ctxs.Contexts = append(ctxs.Contexts, new)

				return ctxs
			}(),
			Run: createHelperFunc(current(ctxNew())),
		},
		{
			Name:        "create duplicate context Name fails",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     errContextNamesAreUnique,
			want:        nil,
			Run: createHelperFunc(&Context{
				Name:     ctx1().Name,
				APIToken: "token123",
			}),
		},
		{
			Name:        "create context without token fails",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     fmt.Errorf(errMsgBlankContextField, "APIToken"),
			want:        nil,
			Run: createHelperFunc(&Context{
				Name:     "notoken",
				APIToken: "",
			}),
		},
	}

	managerTest(t, tests)
}

func TestContextManager_Update(t *testing.T) {
	updateHelperFunc := func(rq *ContextUpdateRequest) func(*testing.T, *ContextManager) (*contexts, error) {
		return func(t *testing.T, manager *ContextManager) (*contexts, error) {
			_, err := manager.Update(rq)
			if err != nil {
				return nil, err
			}
			return manager.getContexts()
		}
	}

	tests := []ManagerTestCase[*contexts]{
		{
			Name:        "update existing context (all fields)",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     nil,
			want: func() *contexts {
				want := contextsActiveUnsetCurrentUnset()
				want.Contexts[0].APIURL = pointer.Pointer("newAPIURL")
				want.Contexts[0].APIToken = "newAPIToken"
				want.Contexts[0].DefaultProject = "newProject"
				want.Contexts[0].Timeout = pointer.Pointer(time.Duration(100))
				want.Contexts[0].Provider = "newProvider"
				return want
			}(),
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:           ctx1().Name,
				APIURL:         pointer.Pointer("newAPIURL"),
				APIToken:       pointer.Pointer("newAPIToken"),
				DefaultProject: pointer.Pointer("newProject"),
				Timeout:        pointer.Pointer(time.Duration(100)),
				Provider:       pointer.Pointer("newProvider"),
			}),
		},
		{
			Name:        "update with activate flag",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsActiveSetCurrentUnset()
				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, ctx3().Name
				ctxs.Contexts[2].IsCurrent = true
				return ctxs
			}(),
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:     ctx3().Name,
				Activate: true,
			}),
		},
		{
			Name:        "update non-existent context",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     fmt.Errorf(errMsgContextNotFound, "nonexistent"),
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:           "nonexistent",
				DefaultProject: pointer.Pointer("foo"),
			}),
		},
		{
			Name:        "update current context without Name",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsActiveSetCurrentSet()
				ctxs.Contexts[0].Provider = "foo"
				return ctxs
			}(),
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:     "",
				Provider: pointer.Pointer("foo"),
			}),
		},
		{
			Name:        "fail with no current and no Name",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     errNoActiveContext,
			want:        nil,
			Run: updateHelperFunc(&ContextUpdateRequest{
				Name:     "",
				Provider: pointer.Pointer("foo"),
			}),
		},
	}

	managerTest(t, tests)
}

func TestContextManager_Delete(t *testing.T) {
	deleteHelperFunc := func(name string) func(*testing.T, *ContextManager) (*contexts, error) {
		return func(t *testing.T, manager *ContextManager) (*contexts, error) {
			_, err := manager.Delete(name)
			if err != nil {
				return nil, err
			}
			return manager.getContexts()
		}
	}

	tests := []ManagerTestCase[*contexts]{
		{
			Name:        "delete existing context",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     nil,
			want: &contexts{
				Contexts: []*Context{ctx1(), ctx2()},
			},
			Run: deleteHelperFunc(ctx3().Name),
		},
		{
			Name:        "delete active context",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsActiveSetCurrentUnset()
				ctxs.Contexts = ctxs.Contexts[1:]
				ctxs.CurrentContext = ""
				return ctxs
			}(),
			Run: deleteHelperFunc(ctx1().Name),
		},
		{
			Name:        "delete previous context",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want: &contexts{
				CurrentContext: ctx1().Name,
				Contexts:       []*Context{current(ctx1()), ctx3()},
			},
			Run: deleteHelperFunc(ctx2().Name),
		},
		{
			Name:        "delete non-existent context",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     fmt.Errorf(errMsgContextNotFound, "nonexistent"),
			want:        nil,
			Run:         deleteHelperFunc("nonexistent"),
		},
	}

	managerTest(t, tests)
}

func TestContexts_Validate(t *testing.T) {
	tests := []struct {
		Name    string
		ctxs    *contexts
		wantErr error
	}{
		{
			Name:    "valid contexts",
			ctxs:    contextsActiveUnsetCurrentUnset(),
			wantErr: nil,
		},
		{
			Name: "duplicate Names",
			ctxs: &contexts{
				Contexts: []*Context{
					{Name: "ctx1", APIToken: "token1"},
					{Name: "ctx1", APIToken: "token2"},
				},
			},
			wantErr: errContextNamesAreUnique,
		},
		{
			Name: "blank Name",
			ctxs: &contexts{
				Contexts: []*Context{
					{Name: "", APIToken: "token1"},
				},
			},
			wantErr: fmt.Errorf(errMsgBlankContextField, "Name"),
		},
		{
			Name: "blank token",
			ctxs: &contexts{
				Contexts: []*Context{
					{Name: "ctx1", APIToken: ""},
				},
			},
			wantErr: fmt.Errorf(errMsgBlankContextField, "APIToken"),
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
func TestContextManager_writeContexts(t *testing.T) {
	tests := []struct {
		Name          string
		InputContexts *contexts
		WantErr       error
		ValidateFile  func(t *testing.T, fs afero.Fs, configPath string)
	}{
		{
			Name:          "write valid contexts",
			InputContexts: contextsActiveSetCurrentUnset(),
			WantErr:       nil,
			ValidateFile: func(t *testing.T, fs afero.Fs, configPath string) {
				exists, err := afero.Exists(fs, configPath)
				require.NoError(t, err)
				require.True(t, exists, "config file should exist")

				content, err := afero.ReadFile(fs, configPath)
				require.NoError(t, err)

				var ctxs contexts
				err = yaml.Unmarshal(content, &ctxs)
				require.NoError(t, err)
				require.Equal(t, "ctx1", ctxs.CurrentContext)
				require.Equal(t, "ctx2", ctxs.PreviousContext)
				require.Len(t, ctxs.Contexts, 3)
			},
		},
		{
			Name:          "write empty contexts",
			InputContexts: &contexts{},
			WantErr:       nil,
			ValidateFile: func(t *testing.T, fs afero.Fs, configPath string) {
				content, err := afero.ReadFile(fs, configPath)
				require.NoError(t, err)

				var ctxs contexts
				err = yaml.Unmarshal(content, &ctxs)
				require.NoError(t, err)
				require.Empty(t, ctxs.CurrentContext)
				require.Empty(t, ctxs.Contexts)
			},
		},
		{
			Name: "fail on duplicate context names",
			InputContexts: &contexts{
				Contexts: []*Context{
					{Name: "duplicate", APIToken: "token1"},
					{Name: "duplicate", APIToken: "token2"},
				},
			},
			WantErr: errContextNamesAreUnique,
		},
		{
			Name: "fail on blank context name",
			InputContexts: &contexts{
				Contexts: []*Context{
					{Name: "", APIToken: "token1"},
				},
			},
			WantErr: fmt.Errorf(errMsgBlankContextField, "Name"),
		},
		{
			Name: "fail on blank API token",
			InputContexts: &contexts{
				Contexts: []*Context{
					{Name: "ctx1", APIToken: ""},
				},
			},
			WantErr: fmt.Errorf(errMsgBlankContextField, "APIToken"),
		},
		{
			Name:          "create config directory if in default path",
			InputContexts: contextsActiveUnsetCurrentUnset(),
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
			InputContexts: &contexts{
				CurrentContext: "ctx2",
				Contexts: []*Context{
					ctx3(),
					ctx1(),
					ctx2(),
				},
			},
			WantErr: nil,
			ValidateFile: func(t *testing.T, fs afero.Fs, configPath string) {
				content, err := afero.ReadFile(fs, configPath)
				require.NoError(t, err)

				var ctxs contexts
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
			InputContexts: &contexts{
				CurrentContext: "ctx3",
				Contexts: []*Context{
					ctx3(), // Has APIURL, Provider, etc.
				},
			},
			WantErr: nil,
			ValidateFile: func(t *testing.T, fs afero.Fs, configPath string) {
				content, err := afero.ReadFile(fs, configPath)
				require.NoError(t, err)

				var ctxs contexts
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

			err = manager.writeContexts(tt.InputContexts)

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

func TestContextManager_getContexts(t *testing.T) {
	tests := []ManagerTestCase[*contexts]{
		{
			Name:        "read existing contexts && IsCurrent is set",
			FileContent: contextsActiveSetCurrentUnset(),
			wantErr:     nil,
			want:        contextsActiveSetCurrentSet(),
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				return manager.getContexts()
			},
		},
		{
			Name:        "read empty config returns empty contexts",
			FileContent: nil,
			wantErr:     nil,
			want: &contexts{
				Contexts: []*Context{},
			},
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				return manager.getContexts()
			},
		},
		{
			Name:        "read contexts without active context",
			FileContent: contextsActiveUnsetCurrentUnset(),
			wantErr:     nil,
			want:        contextsActiveUnsetCurrentUnset(),
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				return manager.getContexts()
			},
		},
		{
			Name: "preserve all context fields",
			FileContent: &contexts{
				CurrentContext: ctx3().Name,
				Contexts: []*Context{
					ctx3(), // Has all optional fields
				},
			},
			wantErr: nil,
			want: &contexts{
				CurrentContext: ctx3().Name,
				Contexts:       []*Context{current(ctx3())},
			},
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				return manager.getContexts()
			},
		},
		{ // TODO do we need to fail here?
			Name: "handle current context not in list",
			FileContent: &contexts{
				CurrentContext: "nonexistent",
				Contexts:       ctxList(),
			},
			wantErr: nil,
			want: &contexts{
				CurrentContext: "nonexistent",
				Contexts:       ctxList(),
			},
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				return manager.getContexts()
			},
		},
	}

	managerTest(t, tests)
}

func TestContextManager_writeContexts_getContexts_RoundTrip(t *testing.T) {
	helperFunc := func(ctxs *contexts) func(*testing.T, *ContextManager) (*contexts, error) {
		return func(t *testing.T, manager *ContextManager) (*contexts, error) {
			err := manager.writeContexts(ctxs)
			if err != nil {
				return nil, err
			}
			return manager.getContexts()
		}
	}
	tests := []ManagerTestCase[*contexts]{
		{
			Name:        "round trip with active context",
			FileContent: nil,
			wantErr:     nil,
			want:        contextsActiveSetCurrentSet(),
			Run:         helperFunc(contextsActiveSetCurrentUnset()),
		},
		{
			Name:        "round trip without active context",
			FileContent: nil,
			wantErr:     nil,
			want:        contextsActiveUnsetCurrentUnset(),
			Run:         helperFunc(contextsActiveUnsetCurrentUnset()),
		},
		{
			Name:        "round trip with all optional fields",
			FileContent: nil,
			wantErr:     nil,
			want: &contexts{
				CurrentContext:  ctx3().Name,
				PreviousContext: ctx1().Name,
				Contexts:        []*Context{current(ctx3()), ctx1()},
			},
			Run: helperFunc(&contexts{
				CurrentContext:  ctx3().Name,
				PreviousContext: ctx1().Name,
				Contexts:        []*Context{ctx3(), ctx1()},
			}),
		},
	}

	managerTest(t, tests)
}

// Console tests below

type consoleTestCase[T any] struct {
	Name        string
	FileContent *contexts
	Args        []string
	Setup       func(t *testing.T, cmd *cobra.Command) error
	wantErr     error
	wantOut     string
	want        T
}

func consoleTest[T any](t *testing.T, tests []consoleTestCase[T]) {
	for _, test := range tests {
		consoleTestOne(t, test)
	}
}

func consoleTestOne[T any](t *testing.T, tt consoleTestCase[T]) {
	t.Run(tt.Name, func(t *testing.T) {
		fs, configDir := setupFs(t)
		mgr := NewContextManager(&ContextConfig{
			BinaryName:      os.Args[0],
			ConfigDirName:   configDir,
			ConfigName:      "config.yaml",
			Fs:              fs,
			Out:             io.Discard,
			ListPrinter:     func() printers.Printer { return printers.NewYAMLPrinter() },
			DescribePrinter: func() printers.Printer { return printers.NewYAMLPrinter() },
		})

		if tt.FileContent == nil {
			tt.FileContent = &contexts{}
		}
		require.NoError(t, mgr.writeContexts(tt.FileContent))

		buf := &bytes.Buffer{}
		cmd := getNewContextCmd(fs, io.Writer(buf), configDir)

		cmd.SetArgs(tt.Args)
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true

		if tt.Setup != nil {
			err := tt.Setup(t, cmd)
			require.NoError(t, err)
		}

		err := cmd.Execute()

		if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
			t.Errorf("error diff (+got -want):\n %s", diff)
			return
		}
		if diff := cmp.Diff(tt.wantOut, buf.String()); diff != "" {
			t.Errorf("Diff = %s", diff)
		}

		result, err := mgr.getContexts()
		require.NoError(t, err)
		if diff := cmp.Diff(tt.want, result); diff != "" {
			t.Errorf("Diff = %s", diff)
		}
	})
}

func newPrinterFromCLI(c *ContextConfig) printers.Printer {
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
	c := &ContextConfig{
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
	tests := []consoleTestCase[*contexts]{
		{
			Name:        "switch to existing context",
			FileContent: contextsActiveSetCurrentUnset(),
			Args:        []string{ctx3().Name},
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsActiveSetCurrentUnset()
				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, ctx3().Name
				ctxs.Contexts[2].IsCurrent = true
				return ctxs
			}(),
			wantOut: fmt.Sprintf("✔ Switched context to \"%s\"\n", ctx3().Name),
		},
		{
			Name:        "switch to the same context",
			FileContent: contextsActiveSetCurrentUnset(),
			Args:        []string{ctx1().Name},
			wantErr:     nil,
			want:        contextsActiveSetCurrentSet(),
			wantOut:     fmt.Sprintf("✔ Context \"%s\" is already active\n", ctx1().Name),
		},
		{
			Name:        "switch to previous context using dash",
			FileContent: contextsActiveSetCurrentUnset(),
			Args:        []string{"-"},
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsActiveSetCurrentUnset()
				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, ctxs.PreviousContext
				ctxs.Contexts[1].IsCurrent = true
				return ctxs
			}(),
			wantOut: fmt.Sprintf("✔ Switched context to \"%s\"\n", ctx2().Name),
		},
		{
			Name:        "switch to previous when none exists",
			FileContent: contextsActiveUnsetCurrentUnset(),
			Args:        []string{"-"},
			wantErr:     errNoPreviousContext,
			want:        contextsActiveUnsetCurrentUnset(),
		},
		{
			Name:        "switch to non-existent context",
			FileContent: contextsActiveSetCurrentUnset(),
			Args:        []string{"nonexistent"},
			wantErr:     fmt.Errorf(errMsgContextNotFound, "nonexistent"),
			want:        contextsActiveSetCurrentSet(),
		},
		{
			Name: "switch to previous when no context is active",
			FileContent: func() *contexts {
				ctxs := contextsActiveSetCurrentUnset()
				ctxs.CurrentContext = ""
				return ctxs
			}(),
			Args:    []string{"-"},
			wantErr: nil,
			want: func() *contexts {
				ctxs := contextsActiveSetCurrentUnset()
				ctxs.PreviousContext, ctxs.CurrentContext = "", ctx2().Name
				ctxs.Contexts[1].IsCurrent = true
				return ctxs
			}(),
			wantOut: fmt.Sprintf("✔ Switched context to \"%s\"\n", ctx2().Name),
		},
	}

	consoleTest(t, tests)
}
