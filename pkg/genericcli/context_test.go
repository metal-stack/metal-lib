package genericcli

import (
	"fmt"
	"io"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
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
		Name:     "ctx3",
		APIURL:   pointer.Pointer("http://foo.bar"),
		APIToken: "token3",
		Provider: "foo",
	}
}

func ctxNew() *Context {
	return &Context{
		Name:     "ctxNew",
		APIToken: "tokenNew",
	}
}

// WARNING these methods are solely to write config file so IsCurrent is not set and defaults to false!
func ctxList() []*Context {
	return []*Context{
		ctx1(),
		ctx2(),
		ctx3(),
	}
}

func contextsNoActiveCtx() *contexts {
	return &contexts{
		CurrentContext:  "",
		PreviousContext: "",
		Contexts:        ctxList(),
	}
}

func contextsWithActiveCtx() *contexts {
	list := ctxList()
	return &contexts{
		CurrentContext:  list[0].Name,
		PreviousContext: list[1].Name,
		Contexts:        list,
	}
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
			FileContent: contextsNoActiveCtx(),
			wantErr:     nil,
			want:        ctx1(),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get(ctx1().Name)
			},
		},
		{
			Name:        "get non-existent context",
			FileContent: contextsNoActiveCtx(),
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
			FileContent: contextsWithActiveCtx(),
			wantErr:     nil,
			want: func() *Context {
				ctx := ctx1()
				ctx.IsCurrent = true
				return ctx
			}(),
			Run: func(t *testing.T, manager *ContextManager) (*Context, error) {
				return manager.Get(ctx1().Name)
			},
		},
	}

	managerTest(t, tests)
}

func TestContextManager_Create(t *testing.T) {
	tests := []ManagerTestCase[*contexts]{
		{
			Name:        "first context auto-activates",
			FileContent: nil,
			wantErr:     nil,
			want: func() *contexts {
				ctx := ctx1()
				ctx.IsCurrent = true
				return &contexts{
					CurrentContext:  ctx.Name,
					PreviousContext: "",
					Contexts:        []*Context{ctx},
				}
			}(),
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				_, err := manager.Create(ctx1())
				if err != nil {
					return nil, err
				}
				return manager.getContexts()
			},
		},
		{
			Name:        "create context with activate flag",
			FileContent: contextsWithActiveCtx(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsWithActiveCtx()
				new := ctxNew()
				new.IsCurrent = true

				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, new.Name
				ctxs.Contexts = append(ctxs.Contexts, new)

				return ctxs
			}(),
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				new := ctxNew()
				new.IsCurrent = true
				_, err := manager.Create(new)
				if err != nil {
					return nil, err
				}
				return manager.getContexts()
			},
		},
		{
			Name:        "create duplicate context Name fails",
			FileContent: contextsWithActiveCtx(),
			wantErr:     errContextNamesAreUnique,
			want:        nil,
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				_, err := manager.Create(&Context{
					Name:     ctx1().Name,
					APIToken: "token123",
				})
				if err != nil {
					return nil, err
				}
				return manager.getContexts()
			},
		},
		{
			Name:        "create context without token fails",
			FileContent: contextsWithActiveCtx(),
			wantErr:     fmt.Errorf(errMsgBlankContextField, "APIToken"),
			want:        nil,
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				_, err := manager.Create(&Context{
					Name:     "notoken",
					APIToken: "",
				})
				if err != nil {
					return nil, err
				}
				return manager.getContexts()
			},
		},
	}

	managerTest(t, tests)
}

func TestContextManager_Update(t *testing.T) {
	tests := []ManagerTestCase[*contexts]{
		{
			Name:        "update existing context",
			FileContent: contextsNoActiveCtx(),
			wantErr:     nil,
			want: func() *contexts {
				want := contextsNoActiveCtx()
				want.Contexts[0].DefaultProject = "new-project"
				return want
			}(),
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				_, err := manager.Update(&ContextUpdateRequest{
					Name:           ctx1().Name,
					DefaultProject: pointer.Pointer("new-project"),
				})
				if err != nil {
					return nil, err
				}
				return manager.getContexts()
			},
		},
		{
			Name:        "update with activate flag",
			FileContent: contextsWithActiveCtx(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsWithActiveCtx()
				ctxs.PreviousContext, ctxs.CurrentContext = ctxs.CurrentContext, ctx3().Name
				ctxs.Contexts[2].IsCurrent = true
				return ctxs
			}(),
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				_, err := manager.Update(&ContextUpdateRequest{
					Name:     ctx3().Name,
					Activate: true,
				})
				if err != nil {
					return nil, err
				}
				return manager.getContexts()
			},
		},
		{
			Name:        "update non-existent context",
			FileContent: contextsWithActiveCtx(),
			wantErr:     fmt.Errorf(errMsgContextNotFound, "nonexistent"),
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				_, err := manager.Update(&ContextUpdateRequest{
					Name:           "nonexistent",
					DefaultProject: pointer.Pointer("foo"),
				})
				if err != nil {
					return nil, err
				}
				return manager.getContexts()
			},
		},
		{
			Name:        "update current context without Name",
			FileContent: contextsWithActiveCtx(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsWithActiveCtx()
				ctxs.Contexts[0].IsCurrent = true
				ctxs.Contexts[0].Provider = "foo"
				return ctxs
			}(),
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				_, err := manager.Update(&ContextUpdateRequest{
					Name:     "",
					Provider: pointer.Pointer("foo"),
				})
				if err != nil {
					return nil, err
				}
				return manager.getContexts()
			},
		},

		{
			Name:        "fail with no current and no Name",
			FileContent: contextsNoActiveCtx(),
			wantErr:     errNoActiveContext,
			want:        nil,
			Run: func(t *testing.T, manager *ContextManager) (*contexts, error) {
				_, err := manager.Update(&ContextUpdateRequest{
					Name:     "",
					Provider: pointer.Pointer("foo"),
				})
				if err != nil {
					return nil, err
				}
				return manager.getContexts()
			},
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
			FileContent: contextsNoActiveCtx(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsNoActiveCtx()
				ctxs.Contexts = ctxs.Contexts[:2] //TODO
				return ctxs
			}(),
			Run: deleteHelperFunc(ctx3().Name),
		},
		{
			Name:        "delete active context",
			FileContent: contextsWithActiveCtx(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsWithActiveCtx()
				ctxs.Contexts = ctxs.Contexts[1:]
				ctxs.CurrentContext = ""
				return ctxs
			}(),
			Run: deleteHelperFunc(ctx1().Name),
		},
		{
			Name:        "delete previous context",
			FileContent: contextsWithActiveCtx(),
			wantErr:     nil,
			want: func() *contexts {
				ctxs := contextsWithActiveCtx()
				ctxs.Contexts = []*Context{ctx1(), ctx3()}
				ctxs.Contexts[0].IsCurrent = true
				ctxs.PreviousContext = ""
				return ctxs
			}(),
			Run: deleteHelperFunc(ctx2().Name),
		},
		{
			Name:        "delete non-existent context",
			FileContent: contextsNoActiveCtx(),
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
			ctxs:    contextsNoActiveCtx(),
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
