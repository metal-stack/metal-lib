package genericcli

import (
	"os"
	"os/exec"

	"gopkg.in/yaml.v2"
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

	err = os.WriteFile(tmpfile.Name(), raw, os.ModePerm)
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

	return a.UpdateFromFile(generic, tmpfile.Name())
}
