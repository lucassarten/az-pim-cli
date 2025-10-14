/*
Copyright © 2023 netr0m <netr0m@pm.me>
*/
package cmd

import (
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/briandowns/spinner"
	"github.com/manifoldco/promptui"
	"github.com/netr0m/az-pim-cli/pkg/pim"
	"github.com/netr0m/az-pim-cli/pkg/utils"
	"github.com/spf13/cobra"
)

var name string
var prefix string
var roleName string
var duration int
var startDate string
var startTime string
var reason string
var ticketSystem string
var ticketNumber string
var dryRun bool
var validateOnly bool
var resources []string
var roles []string

var selectTemplate = &promptui.SelectTemplates{
	Active:   `➤ {{ . | cyan }}`,
	Inactive: `  {{ . | cyan }}`,
	Selected: `✓ {{ . | green }}`,
}

var promptTemplate = &promptui.PromptTemplates{
	Prompt:  `{{ . }} `,
	Valid:   `{{ . | cyan }} `,
	Invalid: `{{ . | red }} `,
	Success: `✓ {{ . | green }} `,
}

var activateCmd = &cobra.Command{
	Use:     "activate",
	Aliases: []string{"a", "ac", "act"},
	Short:   "Send a request to Azure PIM to activate a role assignment",
	Run:     func(cmd *cobra.Command, args []string) {},
}

var resourceSearcher = func(input string, index int) bool {
	name := strings.ToLower(resources[index])
	input = strings.ToLower(input)

	return strings.Contains(name, input)
}

var roleSearcher = func(input string, index int) bool {
	name := strings.ToLower(roles[index])
	input = strings.ToLower(input)

	return strings.Contains(name, input)
}

var s = spinner.New(spinner.CharSets[11], 100*time.Millisecond, spinner.WithWriter(os.Stderr))

var activateResourceCmd = &cobra.Command{
	Use:     "resource",
	Aliases: []string{"r", "res", "resource", "resources", "sub", "subs", "subscriptions"},
	Short:   "Sends a request to Azure PIM to activate the given resource (azure resources)",
	Run: func(cmd *cobra.Command, args []string) {
		token := pim.GetAccessToken(pim.AZ_PIM_SCOPE, pim.AzureClient{})
		subjectId := pim.GetUserInfo(token).ObjectId

		s.Start()
		eligibleResourceAssignments := pim.GetEligibleResourceAssignments(token, pim.AzureClient{})
		eligibleResourceToRoles := utils.GetEligibleResources(eligibleResourceAssignments)
		resources = utils.GetMapKeys(eligibleResourceToRoles)
		s.Stop()

		if name == "" && prefix == "" {
			prompt := promptui.Select{
				Label:             "Select Resource",
				Items:             resources,
				Templates:         selectTemplate,
				Searcher:          resourceSearcher,
				Size:              10,
				StartInSearchMode: true,
			}
			idxResource, _, err := prompt.Run()
			if err != nil {
				log.Fatalf("Prompt aborted: %s", err.Error())
			}
			name = resources[idxResource]
		}

		if roleName == "" {
			roles = eligibleResourceToRoles[name]

			rolePrompt := promptui.Select{
				Label:             "Select Role",
				Items:             roles,
				Templates:         selectTemplate,
				Searcher:          roleSearcher,
				Size:              10,
				StartInSearchMode: true,
			}
			idxRole, _, err := rolePrompt.Run()
			if err != nil {
				log.Fatalf("Prompt aborted: %s", err.Error())
			}
			roleName = eligibleResourceToRoles[name][idxRole]
		}

		if reason == pim.DEFAULT_REASON {
			reasonPrompt := promptui.Prompt{
				Label:   "Reason:",
				Default: reason,
				Templates: promptTemplate,
			}
			result, err := reasonPrompt.Run()
			if err != nil {
				log.Fatalf("Prompt aborted: %s", err.Error())
			}
			reason = result
		}

		resourceAssignment := utils.GetResourceAssignment(name, prefix, roleName, eligibleResourceAssignments)
		scope, assignmentRequest := pim.CreateResourceAssignmentRequest(subjectId, resourceAssignment, duration, startDate, startTime, reason, ticketSystem, ticketNumber)

		log.WithFields(log.Fields{
			"role":         resourceAssignment.Properties.ExpandedProperties.RoleDefinition.DisplayName,
			"scope":        resourceAssignment.Properties.ExpandedProperties.Scope.DisplayName,
			"reason":       reason,
			"ticketNumber": ticketNumber,
			"ticketSystem": ticketSystem,
			"duration":     duration,
		}).Info("Requesting activation")

		if dryRun {
			log.Warn("Skipping activation due to '--dry-run'")
			os.Exit(0)
		}
		if validateOnly {
			log.Warn("Running validation only")
			validationSuccessful := pim.ValidateResourceAssignmentRequest(scope, assignmentRequest, token, pim.AzureClient{})
			if validationSuccessful {
				os.Exit(0)
			}
			os.Exit(1)
		}
		requestResponse := pim.RequestResourceAssignment(scope, assignmentRequest, token, pim.AzureClient{})
		log.WithFields(log.Fields{
			"role":   resourceAssignment.Properties.ExpandedProperties.RoleDefinition.DisplayName,
			"scope":  resourceAssignment.Properties.ExpandedProperties.Scope.DisplayName,
			"status": requestResponse.Properties.Status,
		}).Info("Request completed")
	},
}

func activateGovernanceRole(roleType string) {
	if !pim.IsGovernanceRoleType(roleType) {
		log.Fatal("Invalid role type specified.")
	}
	subjectId := pim.GetUserInfo(pimGovernanceRoleToken).ObjectId

	s.Start()
	eligibleAssignments := pim.GetEligibleGovernanceRoleAssignments(roleType, subjectId, pimGovernanceRoleToken, pim.AzureClient{})
	eligibleResourceToRoles := utils.GetEligibleGovernanceRoles(eligibleAssignments)
	resources = utils.GetMapKeys(eligibleResourceToRoles)
	s.Stop()

	if name == "" && prefix == "" {
		prompt := promptui.Select{
			Label:             "Select Resource",
			Items:             resources,
			Templates:         selectTemplate,
			Searcher:          resourceSearcher,
			Size:              10,
			StartInSearchMode: true,
		}
		idxResource, _, err := prompt.Run()
		if err != nil {
			log.Fatalf("Prompt aborted: %s", err.Error())
		}
		name = resources[idxResource]
	}

	if roleName == "" {
		roles = eligibleResourceToRoles[name]

		rolePrompt := promptui.Select{
			Label:             "Select Role",
			Items:             roles,
			Templates:         selectTemplate,
			Searcher:          roleSearcher,
			Size:              10,
			StartInSearchMode: true,
		}
		idxRole, _, err := rolePrompt.Run()
		if err != nil {
			log.Fatalf("Prompt aborted: %s", err.Error())
		}
		roleName = eligibleResourceToRoles[name][idxRole]
	}

	if reason == pim.DEFAULT_REASON {
		reasonPrompt := promptui.Prompt{
			Label:   "Reason:",
			Default: reason,
			Templates: promptTemplate,
		}
		result, err := reasonPrompt.Run()
		if err != nil {
			log.Fatalf("Prompt aborted: %s", err.Error())
		}
		reason = result
	}

	roleAssignment := utils.GetGovernanceRoleAssignment(name, prefix, roleName, eligibleAssignments)
	roleType, assignmentRequest := pim.CreateGovernanceRoleAssignmentRequest(subjectId, roleType, roleAssignment, duration, startDate, startTime, reason, ticketSystem, ticketNumber)

	log.WithFields(log.Fields{
		"role":         roleAssignment.RoleDefinition.DisplayName,
		"scope":        roleAssignment.RoleDefinition.Resource.DisplayName,
		"reason":       reason,
		"ticketNumber": ticketNumber,
		"ticketSystem": ticketSystem,
		"duration":     duration,
	}).Info("Requesting activation")

	if dryRun {
		log.Warn("Skipping activation due to '--dry-run'")
		os.Exit(0)
	}
	if validateOnly {
		log.Warn("Running validation only")
		validationSuccessful := pim.ValidateGovernanceRoleAssignmentRequest(roleType, assignmentRequest, pimGovernanceRoleToken, pim.AzureClient{})
		if validationSuccessful {
			os.Exit(0)
		}
		os.Exit(1)
	}
	requestResponse := pim.RequestGovernanceRoleAssignment(roleType, assignmentRequest, pimGovernanceRoleToken, pim.AzureClient{})
	log.WithFields(log.Fields{
		"role":   roleAssignment.RoleDefinition.DisplayName,
		"scope":  roleAssignment.RoleDefinition.Resource.DisplayName,
		"status": requestResponse.AssignmentState,
	}).Info("Request completed")

}

var activateGroupCmd = &cobra.Command{
	Use:     "group",
	Aliases: []string{"g", "grp", "groups"},
	Short:   "Sends a request to Azure PIM to activate the given group",
	Run: func(cmd *cobra.Command, args []string) {
		activateGovernanceRole(pim.ROLE_TYPE_AAD_GROUPS)
	},
}

var activateEntraRoleCmd = &cobra.Command{
	Use:     "role",
	Aliases: []string{"rl", "role", "roles"},
	Short:   "Sends a request to Azure PIM to activate the given Entra role",
	Run: func(cmd *cobra.Command, args []string) {
		activateGovernanceRole(pim.ROLE_TYPE_ENTRA_ROLES)
	},
}

func init() {
	rootCmd.AddCommand(activateCmd)
	activateCmd.AddCommand(activateResourceCmd)
	activateCmd.AddCommand(activateGroupCmd)
	activateCmd.AddCommand(activateEntraRoleCmd)

	// Flags
	activateCmd.PersistentFlags().StringVarP(&name, "name", "n", "", "The name of the resource to activate")
	activateCmd.PersistentFlags().StringVarP(&prefix, "prefix", "p", "", "The name prefix of the resource to activate (e.g. 'S399'). Alternative to 'name'.")
	activateCmd.PersistentFlags().StringVarP(&roleName, "role", "r", "", "Specify the role to activate, if multiple roles are found for a resource (e.g. 'Owner' and 'Contributor')")
	activateCmd.PersistentFlags().IntVarP(&duration, "duration", "d", pim.DEFAULT_DURATION_MINUTES, "Duration in minutes that the role should be activated for")
	activateCmd.PersistentFlags().StringVar(&startDate, "start-date", "", "Start date for the activation (as DD/MM/YYYY)")
	activateCmd.PersistentFlags().StringVarP(&startTime, "start-time", "s", "", "Start time for the activation (as HH:MM)")
	activateCmd.PersistentFlags().StringVar(&reason, "reason", pim.DEFAULT_REASON, "Reason for the activation")
	activateCmd.PersistentFlags().StringVar(&ticketSystem, "ticket-system", "", "Ticket system for the activation")
	activateCmd.PersistentFlags().StringVarP(&ticketNumber, "ticket-number", "T", "", "Ticket number for the activation")
	activateCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Display the resource that would be activated, without requesting the activation")
	activateCmd.PersistentFlags().BoolVarP(&validateOnly, "validate-only", "v", false, "Send the request to the validation endpoint of Azure PIM, without requesting the activation")

	activateGroupCmd.PersistentFlags().StringVarP(&pimGovernanceRoleToken, "token", "t", "", "An access token for the PIM 'Entra Roles' and 'Groups' API (required). Consult the README for more information.")
	activateGroupCmd.MarkPersistentFlagRequired("token") //nolint:errcheck

	activateEntraRoleCmd.PersistentFlags().StringVarP(&pimGovernanceRoleToken, "token", "t", "", "An access token for the PIM 'Entra Roles' and 'Groups' API (required). Consult the README for more information.")
	activateEntraRoleCmd.MarkPersistentFlagRequired("token") //nolint:errcheck

	activateCmd.MarkFlagsMutuallyExclusive("name", "prefix")

	s.Suffix = " Fetching role assignments..."
	s.Color("bold", "fgBlue")
}
