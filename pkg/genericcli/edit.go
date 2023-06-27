package genericcli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

func (a *GenericCLI[C, U, R]) Edit(args []string) (R, error) {
	var zero R

	id, err := GetExactlyOneArg(args)
	if err != nil {
		return zero, err
	}

	editor, ok := os.LookupEnv("EDITOR")
	if !ok {
		editor = "vi"
	}

	tmpfile, err := afero.TempFile(a.fs, "", "metallib-*.yaml")
	if err != nil {
		return zero, err
	}
	defer func() {
		_ = a.fs.Remove(tmpfile.Name())
	}()

	doc, err := a.crud.Get(id)
	if err != nil {
		return zero, err
	}

	_, _, updateDoc, err := a.crud.Convert(doc)
	if err != nil {
		return zero, err
	}

	raw, err := yaml.Marshal(updateDoc)
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

	equal, err := YamlIsEqual(raw, editedContent)
	if err != nil {
		return zero, err
	}
	if equal {
		return zero, fmt.Errorf("no changes were made, aborting")
	}

	uparser := MultiDocumentYAML[U]{fs: a.fs}
	updateDoc, err = uparser.ReadOne(tmpfile.Name())
	if err != nil {
		return zero, err
	}

	result, err := a.crud.Update(updateDoc)
	if err != nil {
		return zero, fmt.Errorf("error updating entity: %w", err)
	}

	return result, nil
}

func (a *GenericCLI[C, U, R]) EditAndPrint(args []string, p printers.Printer) error {
	result, err := a.Edit(args)
	if err != nil {
		return err
	}

	return p.Print(result)
}
