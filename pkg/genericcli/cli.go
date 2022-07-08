package genericcli

import "fmt"

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

func (a *GenericCLI[C, U, R]) Describe(args []string) (R, error) {
	var zero R

	id, err := GetExactlyOneArg(args)
	if err != nil {
		return zero, err
	}

	resp, err := a.g.Get(id)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *GenericCLI[C, U, R]) DescribeAndPrint(args []string, p Printer) error {
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

	resp, err := a.g.Delete(id)
	if err != nil {
		return zero, err
	}

	return resp, nil
}

func (a *GenericCLI[C, U, R]) DeleteAndPrint(args []string, p Printer) error {
	resp, err := a.Delete(args)
	if err != nil {
		return err
	}

	return p.Print(resp)
}
