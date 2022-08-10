package genericcli

import (
	"fmt"

	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

func GetExactlyOneArg(args []string) (string, error) {
	switch count := len(args); count {
	case 0:
		return "", fmt.Errorf("a single positional arg is required, none was provided")
	case 1:
		return args[0], nil
	default:
		return "", fmt.Errorf("a single positional arg is required, %d were provided", count)
	}
}

func (a *GenericCLI[C, U, R]) List() ([]R, error) {
	resp, err := a.crud.List()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *GenericCLI[C, U, R]) ListAndPrint(p printers.Printer) error {
	resp, err := a.List()
	if err != nil {
		return err
	}

	return p.Print(resp)
}

func (a *GenericCLI[C, U, R]) Describe(args []string) (R, error) {
	var zero R

	id, err := GetExactlyOneArg(args)
	if err != nil {
		return zero, err
	}

	resp, err := a.crud.Get(id)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *GenericCLI[C, U, R]) DescribeAndPrint(args []string, p printers.Printer) error {
	resp, err := a.Describe(args)
	if err != nil {
		return err
	}

	return p.Print(resp)
}

func (a *GenericCLI[C, U, R]) Delete(args []string) (R, error) {
	var zero R

	id, err := GetExactlyOneArg(args)
	if err != nil {
		return zero, err
	}

	resp, err := a.crud.Delete(id)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *GenericCLI[C, U, R]) DeleteAndPrint(args []string, p printers.Printer) error {
	resp, err := a.Delete(args)
	if err != nil {
		return err
	}

	return p.Print(resp)
}

func (a *GenericCLI[C, U, R]) Create(rq C) (R, error) {
	var zero R

	resp, err := a.crud.Create(rq)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *GenericCLI[C, U, R]) CreateAndPrint(rq C, p printers.Printer) error {
	resp, err := a.Create(rq)
	if err != nil {
		return err
	}

	return p.Print(resp)
}

func (a *GenericCLI[C, U, R]) Update(rq U) (R, error) {
	var zero R

	resp, err := a.crud.Update(rq)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *GenericCLI[C, U, R]) UpdateAndPrint(rq U, p printers.Printer) error {
	resp, err := a.Update(rq)
	if err != nil {
		return err
	}

	return p.Print(resp)
}
