package isolated

import (
	"code.cloudfoundry.org/cli/integration/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = FDescribe("service command", func() {
	var serviceInstance string

	BeforeEach(func() {
		serviceInstance = helpers.PrefixedRandomName("si")
	})

	Describe("help", func() {
		Context("when --help flag is set", func() {
			It("displays command usage to output", func() {
				session := helpers.CF("service", "--help")
				Eventually(session).Should(Say("NAME:"))
				Eventually(session).Should(Say("\\s+service - Show service instance info"))
				Eventually(session).Should(Say("USAGE:"))
				Eventually(session).Should(Say("\\s+cf service SERVICE_INSTANCE"))
				Eventually(session).Should(Say("OPTIONS:"))
				Eventually(session).Should(Say("\\s+--guid\\s+Retrieve and display the given service's guid\\. All other output for the service is suppressed\\."))
				Eventually(session).Should(Say("SEE ALSO:"))
				Eventually(session).Should(Say("\\s+service, rename-service, update-service"))
			})
		})
	})

	Context("when the environment is not setup correctly", func() {
		Context("when no API endpoint is set", func() {
			BeforeEach(func() {
				helpers.UnsetAPI()
			})

			It("fails with no API endpoint set message", func() {
				session := helpers.CF("service", serviceInstance)
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No API endpoint set\\. Use 'cf login' or 'cf api' to target an endpoint\\."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when not logged in", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
			})

			It("fails with not logged in message", func() {
				session := helpers.CF("service", serviceInstance)
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("Not logged in\\. Use 'cf login' to log in\\."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when an org is not targeted", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
				helpers.LoginCF()
			})

			It("fails with no targeted org error", func() {
				session := helpers.CF("service", serviceInstance)
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No org targeted, use 'cf target -o ORG' to target an org\\."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when a space is not targeted", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
				helpers.LoginCF()
				helpers.TargetOrg(ReadOnlyOrg)
			})

			It("fails with no targeted space error", func() {
				session := helpers.CF("service", serviceInstance)
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No space targeted, use 'cf target -s SPACE' to target a space\\."))
				Eventually(session).Should(Exit(1))
			})
		})
	})

	Context("when an api is targeted, the user is logged in, and an org and space are targeted", func() {
		var (
			org      string
			space    string
			username string
		)

		BeforeEach(func() {
			org = helpers.NewOrgName()
			space = helpers.NewSpaceName()
			setupCF(org, space)
			username, _ = helpers.GetCredentials()
		})

		AfterEach(func() {
			helpers.QuickDeleteOrg(org)
		})

		Context("when the service instance does not exist", func() {
			It("returns an error and exits 1", func() {
				session := helpers.CF("service", serviceInstance)
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("Service instance %s not found\\.", serviceInstance))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when the service instance exists", func() {
			var (
				service     string
				servicePlan string
				domain      string
			)

			BeforeEach(func() {
				service = helpers.PrefixedRandomName("SERVICE")
				servicePlan = helpers.PrefixedRandomName("SERVICE-PLAN")
				domain = defaultSharedDomain()
			})

			Context("when the service instance is a user provided service instance", func() {
				BeforeEach(func() {
					Eventually(helpers.CF("create-user-provided-service", serviceInstance, "-p", "{}")).Should(Exit(0))
				})

				AfterEach(func() {
					Eventually(helpers.CF("delete-service", serviceInstance, "-f")).Should(Exit(0))
				})

				It("displays service instance info", func() {
					session := helpers.CF("service", serviceInstance)
					Eventually(session).Should(Say("Getting service info for service %s in org %s / space %s as %s", serviceInstance, org, space, username))
					Eventually(session).Should(Exit(0))
				})

				Context("when the --guid flag is provided", func() {
					It("displays the service instance GUID", func() {})
				})
			})

			Context("when the service instance is a managed service instance", func() {
				It("displays service instance info", func() {})

				Context("when the --guid flag is provided", func() {
					It("displays the service instance GUID", func() {})
				})
			})
		})
	})
})
