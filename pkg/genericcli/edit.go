package genericcli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v3"
)

func (a *GenericCLI[C, U, R]) Edit(generic Generic[C, U, R], id string) (R, error) {
	emptyR := new(R)

	editor, ok := os.LookupEnv("EDITOR")
	if !ok {
		editor = "vi"
	}

	tmpfile, err := os.CreateTemp("", "metallib-*.yaml")
	if err != nil {
		return *emptyR, err
	}
	defer os.Remove(tmpfile.Name())

	content, err := generic.Get(id)
	if err != nil {
		return *emptyR, err
	}

	raw, err := yaml.Marshal(content)
	if err != nil {
		return *emptyR, err
	}

	err = afero.WriteFile(a.fs, tmpfile.Name(), raw, 0755)
	if err != nil {
		return *emptyR, err
	}

	editCommand := exec.Command(editor, tmpfile.Name())
	editCommand.Stdout = os.Stdout
	editCommand.Stdin = os.Stdin
	editCommand.Stderr = os.Stderr

	err = editCommand.Run()
	if err != nil {
		return *emptyR, err
	}

	editedContent, err := afero.ReadFile(a.fs, tmpfile.Name())
	if err != nil {
		return *emptyR, err
	}

	if strings.TrimSpace(string(editedContent)) == strings.TrimSpace(string(raw)) {
		return *emptyR, fmt.Errorf("no changes were made")
	}

	return a.UpdateFromFile(generic, tmpfile.Name())
}
