package v2_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"
	"code.cloudfoundry.org/cli/command/commandfakes"
	"code.cloudfoundry.org/cli/command/translatableerror"
	. "code.cloudfoundry.org/cli/command/v2"
	"code.cloudfoundry.org/cli/command/v2/v2fakes"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("service Command", func() {
	var (
		cmd             ServiceCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v2fakes.FakeServiceActor
		binaryName      string
		executeErr      error
	)

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v2fakes.FakeServiceActor)

		cmd = ServiceCommand{
			UI:          testUI,
			Config:      fakeConfig,
			SharedActor: fakeSharedActor,
			Actor:       fakeActor,
		}

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)

		cmd.RequiredArgs.ServiceInstance = "some-service-instance"
	})

	JustBeforeEach(func() {
		executeErr = cmd.Execute(nil)
	})

	Context("when an error is encountered checking if the environment is setup correctly", func() {
		BeforeEach(func() {
			fakeSharedActor.CheckTargetReturns(sharedaction.NotLoggedInError{BinaryName: binaryName})
		})

		It("returns an error", func() {
			Expect(executeErr).To(MatchError(translatableerror.NotLoggedInError{BinaryName: binaryName}))

			Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
			checkTargetedOrgArg, checkTargetedSpaceArg := fakeSharedActor.CheckTargetArgsForCall(0)
			Expect(checkTargetedOrgArg).To(BeTrue())
			Expect(checkTargetedSpaceArg).To(BeTrue())
		})
	})

	Context("when the user is logged in and an org and space are targeted", func() {
		BeforeEach(func() {
			fakeConfig.TargetedOrganizationReturns(configv3.Organization{
				Name: "some-org",
			})
			fakeConfig.TargetedSpaceReturns(configv3.Space{
				GUID: "some-space-guid",
				Name: "some-space",
			})
		})

		Context("when getting the current user fails", func() {
			BeforeEach(func() {
				fakeConfig.CurrentUserReturns(configv3.User{}, errors.New("get-user-error"))
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError("get-user-error"))
				Expect(fakeConfig.CurrentUserCallCount()).To(Equal(1))
			})
		})

		Context("when getting the current user succeeds", func() {
			BeforeEach(func() {
				fakeConfig.CurrentUserReturns(configv3.User{Name: "some-user"}, nil)
			})

			Context("when the '--guid' flag is provided", func() {
				BeforeEach(func() {
					cmd.GUID = true
				})

				Context("when the service instance does not exist", func() {
					BeforeEach(func() {
						fakeActor.GetServiceInstanceByNameAndSpaceReturns(
							v2action.ServiceInstance{},
							v2action.Warnings{"get-service-instance-warning"},
							v2action.ServiceInstanceNotFoundError{
								GUID: "non-existant-service-instance-guid",
								Name: "non-existant-service-instance",
							})
					})

					It("returns ServiceInstanceNotFoundError", func() {
						Expect(executeErr).To(MatchError(translatableerror.ServiceInstanceNotFoundError{
							GUID: "non-existant-service-instance-guid",
							Name: "non-existant-service-instance",
						}))

						Expect(testUI.Out).To(Say("Showing info of service some-service-instance in org some-org / space some-space as some-user\\.\\.\\."))

						Expect(testUI.Err).To(Say("get-service-instance-warning"))

						Expect(fakeActor.GetServiceInstanceByNameAndSpaceCallCount()).To(Equal(1))
						serviceInstanceNameArg, spaceGUIDArg := fakeActor.GetServiceInstanceByNameAndSpaceArgsForCall(0)
						Expect(serviceInstanceNameArg).To(Equal("some-service-instance"))
						Expect(spaceGUIDArg).To(Equal("some-space-guid"))

						Expect(fakeActor.GetServiceInstanceSummaryByNameAndSpaceCallCount()).To(Equal(0))
					})
				})

				Context("when an error is encountered getting the service instance", func() {
					var expectedErr error

					BeforeEach(func() {
						expectedErr = errors.New("get-service-instance-error")
						fakeActor.GetServiceInstanceByNameAndSpaceReturns(
							v2action.ServiceInstance{},
							v2action.Warnings{"get-service-instance-warning"},
							expectedErr,
						)
					})

					It("returns the error", func() {
						Expect(executeErr).To(MatchError(expectedErr))

						Expect(testUI.Out).To(Say("Showing info of service some-service-instance in org some-org / space some-space as some-user\\.\\.\\."))

						Expect(testUI.Err).To(Say("get-service-instance-warning"))

						Expect(fakeActor.GetServiceInstanceByNameAndSpaceCallCount()).To(Equal(1))
						serviceInstanceNameArg, spaceGUIDArg := fakeActor.GetServiceInstanceByNameAndSpaceArgsForCall(0)
						Expect(serviceInstanceNameArg).To(Equal("some-service-instance"))
						Expect(spaceGUIDArg).To(Equal("some-space-guid"))

						Expect(fakeActor.GetServiceInstanceSummaryByNameAndSpaceCallCount()).To(Equal(0))
					})
				})

				Context("when no errors are encountered getting the service instance", func() {
					BeforeEach(func() {
						fakeActor.GetServiceInstanceByNameAndSpaceReturns(
							v2action.ServiceInstance{
								GUID: "some-service-instance-guid",
								Name: "some-service-instance",
							},
							v2action.Warnings{"get-service-instance-warning"},
							nil,
						)
					})

					It("displays the service instance guid", func() {
						Expect(executeErr).ToNot(HaveOccurred())

						Expect(testUI.Out).To(Say("Showing info of service some-service-instance in org some-org / space some-space as some-user\\.\\.\\."))
						Expect(testUI.Out).To(Say("some-service-instance-guid"))
						Expect(testUI.Err).To(Say("get-service-instance-warning"))

						Expect(fakeActor.GetServiceInstanceByNameAndSpaceCallCount()).To(Equal(1))
						serviceInstanceNameArg, spaceGUIDArg := fakeActor.GetServiceInstanceByNameAndSpaceArgsForCall(0)
						Expect(serviceInstanceNameArg).To(Equal("some-service-instance"))
						Expect(spaceGUIDArg).To(Equal("some-space-guid"))

						Expect(fakeActor.GetServiceInstanceSummaryByNameAndSpaceCallCount()).To(Equal(0))
					})
				})
			})

			Context("when the '--guid' flag is not provided", func() {
				Context("when the service instance does not exist", func() {
					BeforeEach(func() {
						fakeActor.GetServiceInstanceSummaryByNameAndSpaceReturns(
							v2action.ServiceInstanceSummary{},
							v2action.Warnings{"get-service-instance-summary-warning"},
							v2action.ServiceInstanceNotFoundError{
								GUID: "non-existant-service-instance-guid",
								Name: "non-existant-service-instance",
							})
					})

					It("returns ServiceInstanceNotFoundError", func() {
						Expect(executeErr).To(MatchError(translatableerror.ServiceInstanceNotFoundError{
							GUID: "non-existant-service-instance-guid",
							Name: "non-existant-service-instance",
						}))

						Expect(testUI.Out).To(Say("Showing info of service some-service-instance in org some-org / space some-space as some-user\\.\\.\\."))

						Expect(testUI.Err).To(Say("get-service-instance-summary-warning"))

						Expect(fakeActor.GetServiceInstanceSummaryByNameAndSpaceCallCount()).To(Equal(1))
						serviceInstanceNameArg, spaceGUIDArg := fakeActor.GetServiceInstanceSummaryByNameAndSpaceArgsForCall(0)
						Expect(serviceInstanceNameArg).To(Equal("some-service-instance"))
						Expect(spaceGUIDArg).To(Equal("some-space-guid"))

						Expect(fakeActor.GetServiceInstanceByNameAndSpaceCallCount()).To(Equal(0))
					})
				})

				Context("when an error is encountered getting the service instance summary", func() {
					var expectedErr error

					BeforeEach(func() {
						expectedErr = errors.New("get-service-instance-summary-error")
						fakeActor.GetServiceInstanceSummaryByNameAndSpaceReturns(
							v2action.ServiceInstanceSummary{},
							v2action.Warnings{"get-service-instance-summary-warning"},
							expectedErr)
					})

					It("returns the error", func() {
						Expect(executeErr).To(MatchError(expectedErr))

						Expect(testUI.Out).To(Say("Showing info of service some-service-instance in org some-org / space some-space as some-user\\.\\.\\."))

						Expect(testUI.Err).To(Say("get-service-instance-summary-warning"))

						Expect(fakeActor.GetServiceInstanceSummaryByNameAndSpaceCallCount()).To(Equal(1))
						serviceInstanceNameArg, spaceGUIDArg := fakeActor.GetServiceInstanceSummaryByNameAndSpaceArgsForCall(0)
						Expect(serviceInstanceNameArg).To(Equal("some-service-instance"))
						Expect(spaceGUIDArg).To(Equal("some-space-guid"))

						Expect(fakeActor.GetServiceInstanceByNameAndSpaceCallCount()).To(Equal(0))
					})
				})

				Context("when no errors are encountered getting the service instance summary", func() {
					Context("when the service instance is a managed service instance", func() {
						BeforeEach(func() {
							fakeActor.GetServiceInstanceSummaryByNameAndSpaceReturns(
								v2action.ServiceInstanceSummary{
									ServiceInstance: v2action.ServiceInstance{
										Name:         "some-service-instance",
										Type:         ccv2.ManagedService,
										Tags:         []string{"tag-1", "tag-2", "tag-3"},
										DashboardURL: "some-dashboard",
									},
									ServicePlan: v2action.ServicePlan{Name: "some-plan"},
									Service: v2action.Service{
										Label:            "some-service",
										Description:      "some-description",
										DocumentationURL: "some-docs-url",
									},
									BoundApplications: []string{"app-1", "app-2", "app-3"},
								},
								v2action.Warnings{"get-service-instance-summary-warning-1", "get-service-instance-summary-warning-2"},
								nil,
							)
						})

						It("displays the service instance summary", func() {
							Expect(executeErr).ToNot(HaveOccurred())

							Expect(testUI.Out).To(Say("Showing info of service some-service-instance in org some-org / space some-space as some-user\\.\\.\\."))
							Expect(testUI.Out).To(Say(""))
							Expect(testUI.Out).To(Say("name:\\s+some-service-instance"))
							Expect(testUI.Out).To(Say("service:\\s+some-service"))
							Expect(testUI.Out).To(Say("bound apps:\\s+app-1, app-2, app-3"))
							Expect(testUI.Out).To(Say("tags:\\s+tag-1, tag-2, tag-3"))
							Expect(testUI.Out).To(Say("plan:\\s+some-plan"))
							Expect(testUI.Out).To(Say("description:\\s+some-description"))
							Expect(testUI.Out).To(Say("documentation:\\s+some-docs-url"))
							Expect(testUI.Out).To(Say("dashboard:\\s+some-dashboard"))

							Expect(testUI.Err).To(Say("get-service-instance-summary-warning-1"))
							Expect(testUI.Err).To(Say("get-service-instance-summary-warning-2"))

							Expect(fakeActor.GetServiceInstanceSummaryByNameAndSpaceCallCount()).To(Equal(1))
							serviceInstanceNameArg, spaceGUIDArg := fakeActor.GetServiceInstanceSummaryByNameAndSpaceArgsForCall(0)
							Expect(serviceInstanceNameArg).To(Equal("some-service-instance"))
							Expect(spaceGUIDArg).To(Equal("some-space-guid"))

							Expect(fakeActor.GetServiceInstanceByNameAndSpaceCallCount()).To(Equal(0))
						})
					})

					Context("when the service instance is an user provided service instance", func() {
						BeforeEach(func() {
							fakeActor.GetServiceInstanceSummaryByNameAndSpaceReturns(
								v2action.ServiceInstanceSummary{
									ServiceInstance: v2action.ServiceInstance{
										Name: "some-service-instance",
										Type: ccv2.UserProvidedService,
									},
									BoundApplications: []string{"app-1", "app-2", "app-3"},
								},
								v2action.Warnings{"get-service-instance-summary-warning-1", "get-service-instance-summary-warning-2"},
								nil,
							)
						})

						It("displays the service instance summary", func() {
							Expect(executeErr).ToNot(HaveOccurred())

							Expect(testUI.Out).To(Say("Showing info of service some-service-instance in org some-org / space some-space as some-user\\.\\.\\."))
							Expect(testUI.Out).To(Say(""))
							Expect(testUI.Out).To(Say("name:\\s+some-service-instance"))
							Expect(testUI.Out).To(Say("service:\\s+user-provided"))
							Expect(testUI.Out).To(Say("bound apps:\\s+app-1, app-2, app-3"))
							Expect(testUI.Out).ToNot(Say("tags:"))
							Expect(testUI.Out).ToNot(Say("plan:"))
							Expect(testUI.Out).ToNot(Say("description:"))
							Expect(testUI.Out).ToNot(Say("documentation:"))
							Expect(testUI.Out).ToNot(Say("dashboard:"))

							Expect(testUI.Err).To(Say("get-service-instance-summary-warning-1"))
							Expect(testUI.Err).To(Say("get-service-instance-summary-warning-2"))

							Expect(fakeActor.GetServiceInstanceSummaryByNameAndSpaceCallCount()).To(Equal(1))
							serviceInstanceNameArg, spaceGUIDArg := fakeActor.GetServiceInstanceSummaryByNameAndSpaceArgsForCall(0)
							Expect(serviceInstanceNameArg).To(Equal("some-service-instance"))
							Expect(spaceGUIDArg).To(Equal("some-space-guid"))

							Expect(fakeActor.GetServiceInstanceByNameAndSpaceCallCount()).To(Equal(0))
						})
					})
				})
			})
		})
	})
})
