package genericcli

import (
	"fmt"

	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/multisort"
	"github.com/metal-stack/metal-lib/pkg/pointer"
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

func GetExactlyNArgs(n int, args []string) ([]string, error) {
	switch {
	case n == 1:
		arg, err := GetExactlyOneArg(args)
		if err != nil {
			return nil, err
		}
		return pointer.WrapInSlice(arg), nil
	case len(args) == n:
		return args, nil
	default:
		return nil, fmt.Errorf("%d positional args are required, %d were provided", n, len(args))
	}
}

func (a *MultiArgGenericCLI[C, U, R]) List(sortKeys ...multisort.Key) ([]R, error) {
	resp, err := a.crud.List()
	if err != nil {
		return nil, err
	}

	if a.sorter != nil {
		if err := a.sorter.SortBy(resp, sortKeys...); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (a *MultiArgGenericCLI[C, U, R]) ListAndPrint(p printers.Printer, sortKeys ...multisort.Key) error {
	resp, err := a.List(sortKeys...)
	if err != nil {
		return err
	}

	return p.Print(resp)
}

func (a *MultiArgGenericCLI[C, U, R]) Describe(id ...string) (R, error) {
	var zero R

	resp, err := a.crud.Get(id...)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *MultiArgGenericCLI[C, U, R]) DescribeAndPrint(p printers.Printer, id ...string) error {
	resp, err := a.Describe(id...)
	if err != nil {
		return err
	}

	return p.Print(resp)
}

func (a *MultiArgGenericCLI[C, U, R]) Delete(id ...string) (R, error) {
	var zero R

	resp, err := a.crud.Delete(id...)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *MultiArgGenericCLI[C, U, R]) DeleteAndPrint(p printers.Printer, id ...string) error {
	resp, err := a.Delete(id...)
	if err != nil {
		return err
	}

	return p.Print(resp)
}

func (a *MultiArgGenericCLI[C, U, R]) Create(rq C) (R, error) {
	var zero R

	resp, err := a.crud.Create(rq)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *MultiArgGenericCLI[C, U, R]) CreateAndPrint(rq C, p printers.Printer) error {
	resp, err := a.Create(rq)
	if err != nil {
		return err
	}

	return p.Print(resp)
}

func (a *MultiArgGenericCLI[C, U, R]) Update(rq U) (R, error) {
	var zero R

	resp, err := a.crud.Update(rq)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *MultiArgGenericCLI[C, U, R]) UpdateAndPrint(rq U, p printers.Printer) error {
	resp, err := a.Update(rq)
	if err != nil {
		return err
	}

	return p.Print(resp)
}
