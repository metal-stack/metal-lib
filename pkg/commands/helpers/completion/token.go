package completion

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
)

func (c *Completion) TokenListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	req := &apiv2.TokenServiceListRequest{}
	resp, err := c.Client.Apiv2().Token().List(c.Ctx, req)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string
	for _, s := range resp.Tokens {
		fmt.Println(s.Uuid)
		names = append(names, s.Uuid+"\t"+s.Description)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completion) TokenProjectRolesCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	methods, err := c.Client.Apiv2().Method().TokenScopedList(c.Ctx, &apiv2.MethodServiceTokenScopedListRequest{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var roles []string

	for project, role := range methods.ProjectRoles {
		roles = append(roles, project+"="+role.String())
	}

	return roles, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completion) TokenTenantRolesCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	methods, err := c.Client.Apiv2().Method().TokenScopedList(c.Ctx, &apiv2.MethodServiceTokenScopedListRequest{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var roles []string

	for tenant, role := range methods.TenantRoles {
		roles = append(roles, tenant+"="+role.String())
	}

	return roles, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completion) TokenAdminRoleCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var roles []string

	for _, role := range apiv2.AdminRole_name {
		roles = append(roles, role)
	}

	return roles, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completion) TokenPermissionsCompletionfunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	methods, err := c.Client.Apiv2().Method().TokenScopedList(c.Ctx, &apiv2.MethodServiceTokenScopedListRequest{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	subject := ""
	if s, _, ok := strings.Cut(toComplete, "="); ok {
		subject = s
	}

	if subject == "" {
		var perms []string

		for _, p := range methods.Permissions {
			perms = append(perms, p.Subject)
		}

		return perms, cobra.ShellCompDirectiveNoFileComp
	}

	// FIXME: completion does not work at this point, investigate why

	var perms []string

	for _, p := range methods.Permissions {
		perms = append(perms, p.Methods...)
	}

	return perms, cobra.ShellCompDirectiveDefault
}
