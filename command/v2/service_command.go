package v2

import (
	"strings"

	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"
	"code.cloudfoundry.org/cli/command"
	"code.cloudfoundry.org/cli/command/flag"
	"code.cloudfoundry.org/cli/command/v2/shared"
)

//go:generate counterfeiter . ServiceActor

type ServiceActor interface {
	GetServiceInstanceByNameAndSpace(name string, spaceGUID string) (v2action.ServiceInstance, v2action.Warnings, error)
	GetServiceInstanceSummaryByNameAndSpace(name string, spaceGUID string) (v2action.ServiceInstanceSummary, v2action.Warnings, error)
}

type ServiceCommand struct {
	RequiredArgs    flag.ServiceInstance `positional-args:"yes"`
	GUID            bool                 `long:"guid" description:"Retrieve and display the given service's guid.  All other output for the service is suppressed."`
	usage           interface{}          `usage:"CF_NAME service SERVICE_INSTANCE"`
	relatedCommands interface{}          `related_commands:"bind-service, rename-service, update-service"`

	UI          command.UI
	Config      command.Config
	SharedActor command.SharedActor
	Actor       ServiceActor
}

func (cmd *ServiceCommand) Setup(config command.Config, ui command.UI) error {
	cmd.UI = ui
	cmd.Config = config
	cmd.SharedActor = sharedaction.NewActor(config, nil)

	ccClient, uaaClient, err := shared.NewClients(config, ui, true)
	if err != nil {
		return err
	}
	cmd.Actor = v2action.NewActor(ccClient, uaaClient, config)

	return nil
}

func (cmd ServiceCommand) Execute(args []string) error {
	err := cmd.SharedActor.CheckTarget(true, true)
	if err != nil {
		return shared.HandleError(err)
	}

	user, err := cmd.Config.CurrentUser()
	if err != nil {
		return shared.HandleError(err)
	}

	cmd.UI.DisplayTextWithFlavor("Showing info of service {{.ServiceInstanceName}} in org {{.OrgName}} / space {{.SpaceName}} as {{.UserName}}...", map[string]interface{}{
		"ServiceInstanceName": cmd.RequiredArgs.ServiceInstance,
		"OrgName":             cmd.Config.TargetedOrganization().Name,
		"SpaceName":           cmd.Config.TargetedSpace().Name,
		"UserName":            user.Name,
	})
	cmd.UI.DisplayNewline()

	if cmd.GUID {
		return cmd.displayServiceInstanceGUID()
	}

	return cmd.displayServiceInstanceSummary()
}

func (cmd ServiceCommand) displayServiceInstanceGUID() error {
	serviceInstance, warnings, err := cmd.Actor.GetServiceInstanceByNameAndSpace(cmd.RequiredArgs.ServiceInstance, cmd.Config.TargetedSpace().GUID)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return shared.HandleError(err)
	}

	cmd.UI.DisplayText(serviceInstance.GUID)
	return nil
}

func (cmd ServiceCommand) displayServiceInstanceSummary() error {
	serviceInstanceSummary, warnings, err := cmd.Actor.GetServiceInstanceSummaryByNameAndSpace(cmd.RequiredArgs.ServiceInstance, cmd.Config.TargetedSpace().GUID)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return shared.HandleError(err)
	}

	var table [][]string
	serviceInstanceName := serviceInstanceSummary.Name
	boundApps := strings.Join(serviceInstanceSummary.BoundApplications, ", ")

	if ccv2.ServiceInstance(serviceInstanceSummary.ServiceInstance).Managed() {
		table = [][]string{
			{cmd.UI.TranslateText("name:"), serviceInstanceName},
			{cmd.UI.TranslateText("service:"), serviceInstanceSummary.Service.Label},
			{cmd.UI.TranslateText("bound apps:"), boundApps},
			{cmd.UI.TranslateText("tags:"), strings.Join(serviceInstanceSummary.Tags, ", ")},
			{cmd.UI.TranslateText("plan:"), serviceInstanceSummary.ServicePlan.Name},
			{cmd.UI.TranslateText("description:"), serviceInstanceSummary.Service.Description},
			{cmd.UI.TranslateText("documentation:"), serviceInstanceSummary.Service.DocumentationURL},
			{cmd.UI.TranslateText("dashboard:"), serviceInstanceSummary.DashboardURL},
		}
	} else {
		table = [][]string{
			{cmd.UI.TranslateText("name:"), serviceInstanceName},
			{cmd.UI.TranslateText("service:"), cmd.UI.TranslateText("user-provided")},
			{cmd.UI.TranslateText("bound apps:"), boundApps},
		}
	}

	cmd.UI.DisplayKeyValueTable("", table, 3)
	return nil
}
