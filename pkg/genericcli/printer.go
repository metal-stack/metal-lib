package genericcli

import (
	"encoding/json"
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

type (
	Printer interface {
		Print(data interface{}) error
	}

	// JSONPrinter returns data in JSON format
	JSONPrinter struct{}

	// YAMLPrinter returns the model in YAML format
	YAMLPrinter struct{}
)

func NewJSONPrinter() *JSONPrinter {
	return &JSONPrinter{}
}

func (_ *JSONPrinter) Print(data interface{}) error {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", string(content))

	return nil
}

func NewYAMLPrinter() *YAMLPrinter {
	return &YAMLPrinter{}
}

func (_ *YAMLPrinter) Print(data interface{}) error {
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Printf("%s", string(content))

	return nil
}
