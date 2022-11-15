package genericcli

import (
	"fmt"
	"strings"

	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/multisort"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type DefaultCmd string

const (
	ListCmd     DefaultCmd = "list"
	DescribeCmd DefaultCmd = "describe"
	CreateCmd   DefaultCmd = "create"
	UpdateCmd   DefaultCmd = "update"
	DeleteCmd   DefaultCmd = "delete"
	ApplyCmd    DefaultCmd = "apply"
	EditCmd     DefaultCmd = "edit"
)

func allCmds() map[DefaultCmd]bool {
	return map[DefaultCmd]bool{
		ListCmd:     true,
		DescribeCmd: true,
		CreateCmd:   true,
		UpdateCmd:   true,
		DeleteCmd:   true,
		ApplyCmd:    true,
		EditCmd:     true,
	}
}

func OnlyCmds(cmds ...DefaultCmd) map[DefaultCmd]bool {
	res := map[DefaultCmd]bool{}

	for _, c := range cmds {
		res[c] = true
	}

	return res
}

// NewCmds can be used to generate a new cobra/viper root cmd with a set of default cmds provided by the generic cli.
func NewCmds[C any, U any, R any](c *CmdsConfig[C, U, R], additionalCmds ...*cobra.Command) *cobra.Command {
	if len(c.OnlyCmds) == 0 {
		c.OnlyCmds = allCmds()
	}
	if c.Sorter != nil {
		c.GenericCLI = c.GenericCLI.WithSorter(c.Sorter)
	}

	Must(c.validate())

	rootCmd := &cobra.Command{
		Use:     c.Singular,
		Short:   fmt.Sprintf("manage %s entities", c.Singular),
		Long:    c.Description,
		Aliases: c.Aliases,
	}

	var cmds []*cobra.Command

	if _, ok := c.OnlyCmds[ListCmd]; ok {
		cmd := &cobra.Command{
			Use:     "list",
			Aliases: []string{"ls"},
			Short:   fmt.Sprintf("list all %s", c.Plural),
			RunE: func(cmd *cobra.Command, args []string) error {
				sortKeys, err := c.ParseSortFlags()
				if err != nil {
					return err
				}

				return c.GenericCLI.ListAndPrint(c.ListPrinter(), sortKeys...)
			},
		}

		if c.Sorter != nil {
			if sortKeys := c.Sorter.AvailableKeys(); len(sortKeys) > 0 {
				cmd.Flags().StringSlice("sort-by", []string{}, fmt.Sprintf("sort by (comma separated) column(s), sort direction can be changed by appending :asc or :desc behind the column identifier. possible values: %s", strings.Join(sortKeys, "|")))
				Must(cmd.RegisterFlagCompletionFunc("sort-by", cobra.FixedCompletions(c.Sorter.AvailableKeys(), cobra.ShellCompDirectiveNoFileComp)))
			}
		}

		if c.ListCmdMutateFn != nil {
			c.ListCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[DescribeCmd]; ok {
		cmd := &cobra.Command{
			Use:     "describe <id>",
			Aliases: []string{"get"},
			Short:   fmt.Sprintf("describes the %s", c.Singular),
			RunE: func(cmd *cobra.Command, args []string) error {
				id, err := GetExactlyOneArg(args)
				if err != nil {
					return err
				}

				return c.GenericCLI.DescribeAndPrint(id, c.DescribePrinter())
			},
			ValidArgsFunction: c.ValidArgsFn,
		}

		if c.DescribeCmdMutateFn != nil {
			c.DescribeCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[CreateCmd]; ok {
		cmd := &cobra.Command{
			Use:   "create",
			Short: fmt.Sprintf("creates the %s", c.Singular),
			RunE: func(cmd *cobra.Command, args []string) error {
				if c.CreateRequestFromCLI != nil && !viper.IsSet("file") {
					rq, err := c.CreateRequestFromCLI()
					if err != nil {
						return err
					}

					return c.GenericCLI.CreateAndPrint(rq, c.DescribePrinter())
				}

				if viper.GetBool("bulk-output") {
					c.GenericCLI = c.GenericCLI.WithBulkPrint()
				}

				return c.GenericCLI.CreateFromFileAndPrint(viper.GetString("file"), c.ListPrinter())
			},
		}

		cmd.Flags().StringP("file", "f", "", c.fileFlagHelpText("create"))
		cmd.Flags().Bool("bulk-output", false, `when creating from file: prints results in a bulk at the end, the results are a list. default is printing results intermediately during creation, which causes single entities to be printed sequentially.`)

		if c.CreateCmdMutateFn != nil {
			c.CreateCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[UpdateCmd]; ok {
		cmd := &cobra.Command{
			Use:   "update",
			Short: fmt.Sprintf("updates the %s", c.Singular),
			RunE: func(cmd *cobra.Command, args []string) error {
				if c.UpdateRequestFromCLI != nil && !viper.IsSet("file") {
					rq, err := c.UpdateRequestFromCLI(args)
					if err != nil {
						return err
					}

					return c.GenericCLI.UpdateAndPrint(rq, c.DescribePrinter())
				}

				if viper.GetBool("bulk-output") {
					c.GenericCLI = c.GenericCLI.WithBulkPrint()
				}

				return c.GenericCLI.UpdateFromFileAndPrint(viper.GetString("file"), c.ListPrinter())
			},
			ValidArgsFunction: c.ValidArgsFn,
		}

		cmd.Flags().StringP("file", "f", "", c.fileFlagHelpText("update"))
		cmd.Flags().Bool("bulk-output", false, `when updating from file: prints results in a bulk at the end, the results are a list. default is printing results intermediately during update, which causes single entities to be printed sequentially.`)

		if c.UpdateCmdMutateFn != nil {
			c.UpdateCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[DeleteCmd]; ok {
		cmd := &cobra.Command{
			Use:     "delete <id>",
			Short:   fmt.Sprintf("deletes the %s", c.Singular),
			Aliases: []string{"destroy", "rm", "remove"},
			RunE: func(cmd *cobra.Command, args []string) error {
				if !viper.IsSet("file") {
					id, err := GetExactlyOneArg(args)
					if err != nil {
						return err
					}

					return c.GenericCLI.DeleteAndPrint(id, c.DescribePrinter())
				}

				return c.GenericCLI.DeleteFromFileAndPrint(viper.GetString("file"), c.ListPrinter())
			},
			ValidArgsFunction: c.ValidArgsFn,
		}

		cmd.Flags().StringP("file", "f", "", c.fileFlagHelpText("delete"))
		cmd.Flags().Bool("bulk-output", false, `when deleting from file: prints results in a bulk at the end, the results are a list. default is printing results intermediately during deletion, which causes single entities to be printed sequentially.`)

		if c.DeleteCmdMutateFn != nil {
			c.DeleteCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[ApplyCmd]; ok {
		cmd := &cobra.Command{
			Use:   "apply",
			Short: fmt.Sprintf("applies one or more %s from a given file", c.Plural),
			RunE: func(cmd *cobra.Command, args []string) error {
				if viper.GetBool("bulk-output") {
					c.GenericCLI = c.GenericCLI.WithBulkPrint()
				}
				return c.GenericCLI.ApplyFromFileAndPrint(viper.GetString("file"), c.ListPrinter())
			},
		}

		cmd.Flags().StringP("file", "f", "", c.fileFlagHelpText("apply"))
		Must(cmd.MarkFlagRequired("file"))
		cmd.Flags().Bool("bulk-output", false, `prints results in a bulk at the end, the results are a list. default is printing results intermediately during apply, which causes single entities to be printed sequentially.`)

		if c.ApplyCmdMutateFn != nil {
			c.ApplyCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[EditCmd]; ok {
		cmd := &cobra.Command{
			Use:   "edit <id>",
			Short: fmt.Sprintf("edit the %s through an editor and update", c.Singular),
			RunE: func(cmd *cobra.Command, args []string) error {
				return c.GenericCLI.EditAndPrint(args, c.DescribePrinter())
			},
			ValidArgsFunction: c.ValidArgsFn,
		}

		if c.EditCmdMutateFn != nil {
			c.EditCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if c.RootCmdMutateFn != nil {
		c.RootCmdMutateFn(rootCmd)
	}

	rootCmd.AddCommand(cmds...)
	rootCmd.AddCommand(additionalCmds...)

	return rootCmd
}

// CmdsConfig provides the configuration for the default commands.
type CmdsConfig[C any, U any, R any] struct {
	GenericCLI *GenericCLI[C, U, R]

	// OnlyCmds defines which default commands to include from the generic cli. if empty, all default commands will be added.
	OnlyCmds map[DefaultCmd]bool

	// BinaryName is the name of the cli binary.
	BinaryName string
	// Singular, Plural is the name of the entity for which the default cmds are generated.
	Singular, Plural string
	// Description described the entity for which the default cmds are generated.
	Description string
	// Aliases provides additional aliases for the root cmd.
	Aliases []string

	// DescribePrinter is the printer that is used for describing the entity. It's a function because printers potentially get intialized later in the game.
	DescribePrinter func() printers.Printer
	// ListPrinter is the printer that is used for listing multiple entities. It's a function because printers potentially get intialized later in the game.
	ListPrinter func() printers.Printer

	// CreateRequestFromCLI if not nil, this function uses the returned create request to create the entity.
	CreateRequestFromCLI func() (C, error)
	// UpdateRequestFromCLI if not nil, this function uses the returned update request to update the entity.
	UpdateRequestFromCLI func(args []string) (U, error)

	// Sorter allows sorting the results of list commands.
	Sorter *multisort.Sorter[R]

	// ValidArgsFn is a completion function that returns the valid command line arguments.
	ValidArgsFn func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

	// MutateFns can be used to customize default commands (adding additional CLI flags or something like that)
	RootCmdMutateFn     func(cmd *cobra.Command)
	ListCmdMutateFn     func(cmd *cobra.Command)
	DescribeCmdMutateFn func(cmd *cobra.Command)
	CreateCmdMutateFn   func(cmd *cobra.Command)
	UpdateCmdMutateFn   func(cmd *cobra.Command)
	DeleteCmdMutateFn   func(cmd *cobra.Command)
	ApplyCmdMutateFn    func(cmd *cobra.Command)
	EditCmdMutateFn     func(cmd *cobra.Command)
}

func (c *CmdsConfig[C, U, R]) ParseSortFlags() (multisort.Keys, error) {
	var keys multisort.Keys

	for _, col := range viper.GetStringSlice("sort-by") {
		col = strings.ToLower(strings.TrimSpace(col))

		var descending bool

		id, directionRaw, found := strings.Cut(col, ":")
		if found {
			switch directionRaw {
			case "asc", "ascending":
				descending = false
			case "desc", "descending":
				descending = true
			default:
				return nil, fmt.Errorf("unsupported sort direction: %s", directionRaw)
			}
		}

		keys = append(keys, multisort.Key{ID: id, Descending: descending})
	}

	return keys, nil
}

func (c *CmdsConfig[C, U, R]) validate() error {
	if c.GenericCLI == nil {
		return fmt.Errorf("generic cli must not be nil, command: %s", c.Singular)
	}
	if c.DescribePrinter == nil {
		return fmt.Errorf("describe must not be nil, command: %s", c.Singular)
	}
	if c.ListPrinter == nil {
		return fmt.Errorf("list printer must not be nil, command: %s", c.Singular)
	}
	if len(c.OnlyCmds) == 0 {
		return fmt.Errorf("included cmds must not be zero length, command: %s", c.Singular)
	}
	if c.BinaryName == "" {
		return fmt.Errorf("binary name must not be empty, command: %s", c.Singular)
	}
	if c.Singular == "" {
		return fmt.Errorf("singular must not be empty, command: %s", c.Singular)
	}
	if c.Plural == "" {
		return fmt.Errorf("plural must not be empty, command: %s", c.Singular)
	}
	if c.Description == "" {
		return fmt.Errorf("description must not be empty, command: %s", c.Singular)
	}

	return nil
}

func (c *CmdsConfig[C, U, R]) fileFlagHelpText(command string) string {
	return fmt.Sprintf(`filename of the create or update request in yaml format, or - for stdin.

Example:
$ %[2]s %[1]s describe %[1]s-1 -o yaml > %[1]s.yaml
$ vi %[1]s.yaml
$ # either via stdin
$ cat %[1]s.yaml | %[2]s %[1]s %[3]s -f -
$ # or via file
$ %[2]s %[1]s %[3]s -f %[1]s.yaml
	`, c.Singular, c.BinaryName, command)
}
