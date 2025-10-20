package completion

import (
	adminv2 "github.com/metal-stack/api/go/metalstack/admin/v2"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/spf13/cobra"
)

func (c *Completion) TenantListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	req := &apiv2.TenantServiceListRequest{}
	resp, err := c.Client.Apiv2().Tenant().List(c.Ctx, req)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string
	for _, t := range resp.Tenants {
		names = append(names, t.Login+"\t"+t.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completion) TenantRoleCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var names []string

	for value, name := range apiv2.TenantRole_name {
		if value == 0 {
			continue
		}

		names = append(names, name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completion) TenantInviteListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projectResp, err := c.Client.Apiv2().Project().Get(c.Ctx, &apiv2.ProjectServiceGetRequest{
		Project: c.Project,
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	resp, err := c.Client.Apiv2().Tenant().InvitesList(c.Ctx, &apiv2.TenantServiceInvitesListRequest{
		Login: projectResp.Project.Tenant,
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string

	for _, invite := range resp.Invites {
		names = append(names, invite.Secret+"\t"+invite.Role.String())
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completion) TenantMemberListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projectResp, err := c.Client.Apiv2().Project().Get(c.Ctx, &apiv2.ProjectServiceGetRequest{
		Project: c.Project,
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	resp, err := c.Client.Apiv2().Tenant().Get(c.Ctx, &apiv2.TenantServiceGetRequest{
		Login: projectResp.Project.Tenant,
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string

	for _, member := range resp.TenantMembers {
		names = append(names, member.Id+"\t"+member.Role.String())
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completion) AdminTenantListCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	req := &adminv2.TenantServiceListRequest{}
	resp, err := c.Client.Adminv2().Tenant().List(c.Ctx, req)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var names []string
	for _, s := range resp.Tenants {
		names = append(names, s.Login+"\t"+s.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
