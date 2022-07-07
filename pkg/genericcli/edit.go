package genericcli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v3"
)

func (a *GenericCLI[C, U, R]) Edit(id string) (R, error) {
	var zero R

	editor, ok := os.LookupEnv("EDITOR")
	if !ok {
		editor = "vi"
	}

	tmpfile, err := os.CreateTemp("", "metallib-*.yaml")
	if err != nil {
		return zero, err
	}
	defer os.Remove(tmpfile.Name())

	content, err := a.g.Get(id)
	if err != nil {
		return zero, err
	}

	raw, err := yaml.Marshal(content)
	if err != nil {
		return zero, err
	}

	err = afero.WriteFile(a.fs, tmpfile.Name(), raw, 0755)
	if err != nil {
		return zero, err
	}

	editCommand := exec.Command(editor, tmpfile.Name())
	editCommand.Stdout = os.Stdout
	editCommand.Stdin = os.Stdin
	editCommand.Stderr = os.Stderr

	err = editCommand.Run()
	if err != nil {
		return zero, err
	}

	editedContent, err := afero.ReadFile(a.fs, tmpfile.Name())
	if err != nil {
		return zero, err
	}

	if strings.TrimSpace(string(editedContent)) == strings.TrimSpace(string(raw)) {
		return zero, fmt.Errorf("no changes were made")
	}

	return a.UpdateFromFile(tmpfile.Name())
}
