package genericcli

import (
	"errors"
	"fmt"
	"io"
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

// CmdsConfig provides the configuration for the default commands.
type CmdsConfig[C any, U any, R any] struct {
	// GenericCLI is the generic CLI used by the cobra commands. this uses only single positional arguments. if you have multiple, use multi arg generic cli.
	GenericCLI *GenericCLI[C, U, R]
	// MultiArgGenericCLI is the generic CLI used by the cobra commands. this can use n positional arguments.
	MultiArgGenericCLI *MultiArgGenericCLI[C, U, R]

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

	// Args defines how many arguments are being used for the entity's id and how they are named, this defaults to ["id"]
	Args []string

	// DescribePrinter is the printer that is used for describing the entity. It's a function because printers potentially get initialized later in the game.
	DescribePrinter func() printers.Printer
	// ListPrinter is the printer that is used for listing multiple entities. It's a function because printers potentially get initialized later in the game.
	ListPrinter func() printers.Printer

	// CreateRequestFromCLI if not nil, this function uses the returned create request to create the entity.
	CreateRequestFromCLI func() (C, error)
	// UpdateRequestFromCLI if not nil, this function uses the returned update request to update the entity.
	UpdateRequestFromCLI func(args []string) (U, error)

	// Sorter allows sorting the results of list commands.
	Sorter *multisort.Sorter[R]

	// ValidArgsFn is a completion function that returns the valid command line arguments.
	ValidArgsFn func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

	// In defines from where input is read, defaults to stdin.
	In io.Reader
	// Out defines to where output is written, defaults to stdout.
	Out io.Writer

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

// NewCmds can be used to generate a new cobra/viper root cmd with a set of default cmds provided by the generic cli.
func NewCmds[C any, U any, R any](c *CmdsConfig[C, U, R], additionalCmds ...*cobra.Command) *cobra.Command {
	if len(c.OnlyCmds) == 0 {
		c.OnlyCmds = allCmds()
	}
	if len(c.Args) == 0 {
		c.Args = []string{"id"}
	}
	if c.GenericCLI != nil {
		c.MultiArgGenericCLI = c.GenericCLI.multiCLI
	}
	if c.Sorter != nil {
		c.MultiArgGenericCLI = c.MultiArgGenericCLI.WithSorter(c.Sorter)
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
				sortKeys, err := ParseSortFlags()
				if err != nil {
					return err
				}

				return c.MultiArgGenericCLI.ListAndPrint(c.ListPrinter(), sortKeys...)
			},
		}

		if c.Sorter != nil {
			AddSortFlag(cmd, c.Sorter)
		}

		if c.ListCmdMutateFn != nil {
			c.ListCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[DescribeCmd]; ok {
		use := "describe"
		for _, arg := range c.Args {
			use += fmt.Sprintf(" <%s>", arg)
		}

		cmd := &cobra.Command{
			Use:     use,
			Aliases: []string{"get"},
			Short:   fmt.Sprintf("describes the %s", c.Singular),
			RunE: func(cmd *cobra.Command, args []string) error {
				id, err := GetExactlyNArgs(len(c.Args), args)
				if err != nil {
					return err
				}

				return c.MultiArgGenericCLI.DescribeAndPrint(c.DescribePrinter(), id...)
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

					return c.MultiArgGenericCLI.CreateAndPrint(rq, c.DescribePrinter())
				}

				p := c.evalBulkFlags()

				return c.MultiArgGenericCLI.CreateFromFileAndPrint(viper.GetString("file"), p())
			},
		}

		c.addFileFlags(cmd)

		if c.CreateCmdMutateFn != nil {
			c.CreateCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[UpdateCmd]; ok {
		use := "update"
		if c.UpdateRequestFromCLI != nil {
			for _, arg := range c.Args {
				use += fmt.Sprintf(" <%s>", arg)
			}
		}

		cmd := &cobra.Command{
			Use:   use,
			Short: fmt.Sprintf("updates the %s", c.Singular),
			RunE: func(cmd *cobra.Command, args []string) error {
				if c.UpdateRequestFromCLI != nil && !viper.IsSet("file") {
					rq, err := c.UpdateRequestFromCLI(args)
					if err != nil {
						return err
					}

					return c.MultiArgGenericCLI.UpdateAndPrint(rq, c.DescribePrinter())
				}

				p := c.evalBulkFlags()

				return c.MultiArgGenericCLI.UpdateFromFileAndPrint(viper.GetString("file"), p())
			},
			ValidArgsFunction: c.ValidArgsFn,
		}

		c.addFileFlags(cmd)

		if c.UpdateCmdMutateFn != nil {
			c.UpdateCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[DeleteCmd]; ok {
		use := "delete"
		for _, arg := range c.Args {
			use += fmt.Sprintf(" <%s>", arg)
		}

		cmd := &cobra.Command{
			Use:     use,
			Short:   fmt.Sprintf("deletes the %s", c.Singular),
			Aliases: []string{"destroy", "rm", "remove"},
			RunE: func(cmd *cobra.Command, args []string) error {
				if !viper.IsSet("file") {
					id, err := GetExactlyNArgs(len(c.Args), args)
					if err != nil {
						return err
					}

					return c.MultiArgGenericCLI.DeleteAndPrint(c.DescribePrinter(), id...)
				}

				p := c.evalBulkFlags()

				return c.MultiArgGenericCLI.DeleteFromFileAndPrint(viper.GetString("file"), p())
			},
			ValidArgsFunction: c.ValidArgsFn,
		}

		c.addFileFlags(cmd)

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
				if !viper.GetBool("skip-security-prompts") {
					c.MultiArgGenericCLI = c.MultiArgGenericCLI.WithBulkSecurityPrompt(c.In, c.Out)
				}

				p := c.evalBulkFlags()

				return c.MultiArgGenericCLI.ApplyFromFileAndPrint(viper.GetString("file"), p())
			},
		}

		c.addFileFlags(cmd)
		Must(cmd.MarkFlagRequired("file"))

		if c.ApplyCmdMutateFn != nil {
			c.ApplyCmdMutateFn(cmd)
		}

		cmds = append(cmds, cmd)
	}

	if _, ok := c.OnlyCmds[EditCmd]; ok {
		use := "edit"
		for _, arg := range c.Args {
			use += fmt.Sprintf(" <%s>", arg)
		}

		cmd := &cobra.Command{
			Use:   use,
			Short: fmt.Sprintf("edit the %s through an editor and update", c.Singular),
			RunE: func(cmd *cobra.Command, args []string) error {
				return c.MultiArgGenericCLI.EditAndPrint(len(c.Args), args, c.DescribePrinter())
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

func ParseSortFlags() (multisort.Keys, error) {
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

func AddSortFlag[R any](cmd *cobra.Command, sorter *multisort.Sorter[R]) {
	if sortKeys := sorter.AvailableKeys(); len(sortKeys) > 0 {
		cmd.Flags().StringSlice("sort-by", []string{}, fmt.Sprintf("sort by (comma separated) column(s), sort direction can be changed by appending :asc or :desc behind the column identifier. possible values: %s", strings.Join(sortKeys, "|")))
		Must(cmd.RegisterFlagCompletionFunc("sort-by", cobra.FixedCompletions(sorter.AvailableKeys(), cobra.ShellCompDirectiveNoFileComp)))
	}
}

func (c *CmdsConfig[C, U, R]) addFileFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("file", "f", "", c.fileFlagHelpText(cmd.Use))
	cmd.Flags().Bool("skip-security-prompts", false, c.skipPromptsFlagText())
	cmd.Flags().Bool("bulk-output", false, c.bulkFlagText())
	cmd.Flags().Bool("timestamps", false, c.bulkTimestampsText())
}

func (c *CmdsConfig[C, U, R]) validate() error {
	if c.MultiArgGenericCLI == nil {
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
	if len(c.Args) < 1 {
		return errors.New("at least one arg for id is required")
	}

	return nil
}

func (c *CmdsConfig[C, U, R]) evalBulkFlags() func() printers.Printer {
	if !viper.GetBool("skip-security-prompts") {
		c.MultiArgGenericCLI = c.MultiArgGenericCLI.WithBulkSecurityPrompt(c.In, c.Out)
	}

	if viper.GetBool("timestamps") {
		c.MultiArgGenericCLI = c.MultiArgGenericCLI.WithTimestamps()
	}

	p := c.DescribePrinter
	if viper.GetBool("bulk-output") {
		p = c.ListPrinter
		c.MultiArgGenericCLI = c.MultiArgGenericCLI.WithBulkPrint()
	}

	return p
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

the file can also contain multiple documents and perform a bulk operation.
	`, c.Singular, c.BinaryName, command)
}

func (c *CmdsConfig[C, U, R]) skipPromptsFlagText() string {
	return "skips security prompt for bulk operations"
}

func (c *CmdsConfig[C, U, R]) bulkFlagText() string {
	return "when used with --file (bulk operation): prints results at the end as a list. default is printing results intermediately during the operation, which causes single entities to be printed in a row."
}

func (c *CmdsConfig[C, U, R]) bulkTimestampsText() string {
	return "when used with --file (bulk operation): prints timestamps in-between the operations"
}
