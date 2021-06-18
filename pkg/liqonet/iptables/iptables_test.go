package iptables

import (
	"fmt"
	"os"
	"strings"

	. "github.com/coreos/go-iptables/iptables"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/klog"

	"github.com/liqotech/liqo/apis/net/v1alpha1"
	"github.com/liqotech/liqo/pkg/consts"
	"github.com/liqotech/liqo/pkg/liqonet/errors"
	"github.com/liqotech/liqo/pkg/liqonet/utils"
)

const (
	clusterID1             = "cluster1"
	invalidValue           = "an invalid value"
	remoteNATPodCIDRValue1 = "10.60.0.0/24"
	remoteNATPodCIDRValue2 = "10.70.0.0/24"
)

var h IPTHandler
var ipt *IPTables
var tep *v1alpha1.TunnelEndpoint

var _ = Describe("iptables", func() {
	BeforeEach(func() {
		var err error
		h, err = NewIPTHandler()
		Expect(err).To(BeNil())
		ipt, err = New()
		Expect(err).To(BeNil())
		tep = forgeValidTEPResource()
	})
	Describe("Init", func() {
		Context("Call func", func() {
			It("should produce no errors and create Liqo chains", func() {
				err := h.Init()
				Expect(err).To(BeNil())

				// Retrieve NAT chains and Filter chains
				natChains, err := ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				filterChains, err := ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				// Check existence of LIQO-POSTROUTING chain
				Expect(natChains).To(ContainElement(liqonetPostroutingChain))
				// Check existence of rule in POSTROUTING
				postRoutingRules, err := h.ListRulesInChain(postroutingChain)
				Expect(err).To(BeNil())
				Expect(postRoutingRules).To(ContainElement(fmt.Sprintf("-j %s", liqonetPostroutingChain)))

				// Check existence of LIQO-PREROUTING chain
				Expect(natChains).To(ContainElement(liqonetPostroutingChain))
				// Check existence of rule in PREROUTING
				preRoutingRules, err := h.ListRulesInChain(preroutingChain)
				Expect(err).To(BeNil())
				Expect(preRoutingRules).To(ContainElement(fmt.Sprintf("-j %s", liqonetPreroutingChain)))

				// Check existence of LIQO-FORWARD chain
				Expect(filterChains).To(ContainElement(liqonetForwardingChain))
				// Check existence of rule in FORWARD
				forwardRules, err := h.ListRulesInChain(forwardChain)
				Expect(err).To(BeNil())
				Expect(forwardRules).To(ContainElement(fmt.Sprintf("-j %s", liqonetForwardingChain)))

				// Check existence of LIQO-INPUT chain
				Expect(filterChains).To(ContainElement(liqonetInputChain))
				// Check existence of rule in INPUT
				inputRules, err := h.ListRulesInChain(inputChain)
				Expect(err).To(BeNil())
				Expect(inputRules).To(ContainElement(fmt.Sprintf("-j %s", liqonetInputChain)))
			})
		})

		Context("Call func twice", func() {
			It("should produce no errors and insert all the rules", func() {
				err := h.Init()
				Expect(err).To(BeNil())

				// Check only POSTROUTING chain and rules

				// Retrieve NAT chains
				natChains, err := ipt.ListChains(natTable)
				Expect(err).To(BeNil())

				// Check existence of LIQO-POSTROUTING chain
				Expect(natChains).To(ContainElement(liqonetPostroutingChain))
				// Check existence of rule in POSTROUTING
				postRoutingRules, err := h.ListRulesInChain(postroutingChain)
				Expect(err).To(BeNil())
				Expect(postRoutingRules).To(ContainElements([]string{
					fmt.Sprintf("-j %s", liqonetPostroutingChain),
				}))
			})
		})
	})

	Describe("EnsureChainRulesPerCluster", func() {
		BeforeEach(func() {
			err := h.Init()
			Expect(err).To(BeNil())
			err = h.EnsureChainsPerCluster(clusterID1)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			err := h.RemoveIPTablesConfigurationPerCluster(tep)
			Expect(err).To(BeNil())
			err = h.Terminate()
			Expect(err).To(BeNil())
		})
		Context(fmt.Sprintf("If all parameters are valid and LocalNATPodCIDR is equal to "+
			"constant value %s in TunnelEndpoint resource", consts.DefaultCIDRValue), func() {
			It(`should add chain rules in POSTROUTING, INPUT, FORWARD but not in PREROUTING`, func() {
				tep.Status.LocalNATPodCIDR = consts.DefaultCIDRValue

				err := h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(BeNil())

				clusterPostRoutingChain := strings.Join([]string{liqonetPostroutingClusterChainPrefix, strings.Split(tep.Spec.ClusterID, "-")[0]}, "")
				clusterForwardChain := strings.Join([]string{liqonetForwardingClusterChainPrefix, strings.Split(tep.Spec.ClusterID, "-")[0]}, "")
				clusterInputChain := strings.Join([]string{liqonetInputClusterChainPrefix, strings.Split(tep.Spec.ClusterID, "-")[0]}, "")

				// Check existence of rule in LIQO-POSTROUTING chain
				postRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)
				expectedRule := fmt.Sprintf("-d %s -j %s", tep.Status.RemoteNATPodCIDR, clusterPostRoutingChain)
				Expect(expectedRule).To(Equal(postRoutingRules[0]))

				// Check existence of rule in LIQO-PREROUTING chain (should not exist)
				preRoutingRules, err := h.ListRulesInChain(liqonetPreroutingChain)
				Expect(preRoutingRules).To(HaveLen(0))

				// Check existence of rule in LIQO-FORWARD chain
				forwardRules, err := h.ListRulesInChain(liqonetForwardingChain)
				expectedRule = fmt.Sprintf("-d %s -j %s", tep.Status.RemoteNATPodCIDR, clusterForwardChain)
				Expect(expectedRule).To(Equal(forwardRules[0]))

				// Check existence of rule in LIQO-INPUT chain
				inputRules, err := h.ListRulesInChain(liqonetInputChain)
				expectedRule = fmt.Sprintf("-d %s -j %s", tep.Status.RemoteNATPodCIDR, clusterInputChain)
				Expect(expectedRule).To(Equal(inputRules[0]))

			})
		})

		Context(fmt.Sprintf("If all parameters are valid and LocalNATPodCIDR is not equal to "+
			"constant value %s in TunnelEndpoint resource", consts.DefaultCIDRValue), func() {
			It(`should add chain rules in POSTROUTING, INPUT, FORWARD and PREROUTING`, func() {
				err := h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(BeNil())

				clusterPostRoutingChain := strings.Join([]string{liqonetPostroutingClusterChainPrefix, strings.Split(tep.Spec.ClusterID, "-")[0]}, "")
				clusterForwardChain := strings.Join([]string{liqonetForwardingClusterChainPrefix, strings.Split(tep.Spec.ClusterID, "-")[0]}, "")
				clusterInputChain := strings.Join([]string{liqonetInputClusterChainPrefix, strings.Split(tep.Spec.ClusterID, "-")[0]}, "")
				clusterPreRoutingChain := strings.Join([]string{liqonetPreroutingClusterChainPrefix, strings.Split(tep.Spec.ClusterID, "-")[0]}, "")

				// Check existence of rule in LIQO-PREROUTING chain
				postRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)
				expectedRule := fmt.Sprintf("-d %s -j %s", tep.Status.RemoteNATPodCIDR, clusterPostRoutingChain)
				Expect(expectedRule).To(Equal(postRoutingRules[0]))

				// Check existence of rule in LIQO-FORWARD chain
				forwardRules, err := h.ListRulesInChain(liqonetForwardingChain)
				expectedRule = fmt.Sprintf("-d %s -j %s", tep.Status.RemoteNATPodCIDR, clusterForwardChain)
				Expect(expectedRule).To(Equal(forwardRules[0]))

				// Check existence of rule in LIQO-INPUT chain
				inputRules, err := h.ListRulesInChain(liqonetInputChain)
				expectedRule = fmt.Sprintf("-d %s -j %s", tep.Status.RemoteNATPodCIDR, clusterInputChain)
				Expect(expectedRule).To(Equal(inputRules[0]))

				// Check existence of rule in LIQO-PREROUTING chain
				preRoutingRules, err := h.ListRulesInChain(liqonetPreroutingChain)
				expectedRule = fmt.Sprintf("-s %s -d %s -j %s", tep.Status.RemoteNATPodCIDR,
					tep.Status.LocalNATPodCIDR, clusterPreRoutingChain)
				Expect(expectedRule).To(Equal(preRoutingRules[0]))
			})
		})

		Context(fmt.Sprintf("If RemoteNATPodCIDR is different from constant value %s "+
			"and is not a valid network in TunnelEndpoint resource", consts.DefaultCIDRValue), func() {
			It("should return a WrongParameter error", func() {
				tep.Status.RemoteNATPodCIDR = invalidValue
				err := h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", remoteNATPodCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})

		Context(fmt.Sprintf("If RemoteNATPodCIDR is equal to constant value %s "+
			"and RemotePodCIDR is not a valid network in TunnelEndpoint resource", consts.DefaultCIDRValue), func() {
			It("should return a WrongParameter error", func() {
				tep.Status.RemoteNATPodCIDR = consts.DefaultCIDRValue
				tep.Spec.PodCIDR = invalidValue
				err := h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", podCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource() // Otherwise RemoveIPTablesConfigurationPerCluster would fail in AfterEach
			})
		})

		Context(fmt.Sprintf("If LocalNATPodCIDR is not equal to constant value %s "+
			"and is not a valid network in TunnelEndpoint resource", consts.DefaultCIDRValue), func() {
			It("should return a WrongParameter error", func() {
				tep.Status.LocalNATPodCIDR = invalidValue
				err := h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", localNATPodCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})

		Context(fmt.Sprintf("If LocalNATPodCIDR is not equal to constant value %s "+
			"and is not a valid network in TunnelEndpoint resource", consts.DefaultCIDRValue), func() {
			It("should return a WrongParameter error", func() {
				tep.Status.LocalNATPodCIDR = invalidValue
				err := h.Init()
				Expect(err).To(BeNil())
				err = h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", localNATPodCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})

		Context("If there are already some rules in chains but they are not in new rules", func() {
			It(`should remove existing rules that are not in the set of new rules and add new rules`, func() {
				err := h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(BeNil())

				clusterPostRoutingChain := strings.Join([]string{liqonetPostroutingClusterChainPrefix, strings.Split(tep.Spec.ClusterID, "-")[0]}, "")

				// Get rule that will be removed
				postRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)
				outdatedRule := postRoutingRules[0]

				// Modify resource
				tep.Status.RemoteNATPodCIDR = remoteNATPodCIDRValue2

				// Second call
				err = h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(BeNil())
				newPostRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)

				// Check if new rule has been added.
				expectedRule := fmt.Sprintf("-d %s -j %s", tep.Status.RemoteNATPodCIDR, clusterPostRoutingChain)
				Expect(expectedRule).To(Equal(newPostRoutingRules[0]))

				// Check if outdated rule has been removed
				Expect(newPostRoutingRules).ToNot(ContainElement(outdatedRule))
			})
		})

		Context("If there are already some rules in chains but they are not in new rules", func() {
			It(`should remove existing rules that are not in the set of new rules and add new rules`, func() {
				err := h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(BeNil())

				clusterPostRoutingChain := strings.Join([]string{liqonetPostroutingClusterChainPrefix, strings.Split(tep.Spec.ClusterID, "-")[0]}, "")

				// Get rule that will be removed
				postRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)
				outdatedRule := postRoutingRules[0]

				// Modify resource
				tep.Status.RemoteNATPodCIDR = remoteNATPodCIDRValue2

				// Second call
				err = h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(BeNil())
				newPostRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)

				// Check if new rule has been added.
				expectedRule := fmt.Sprintf("-d %s -j %s", tep.Status.RemoteNATPodCIDR, clusterPostRoutingChain)
				Expect(expectedRule).To(Equal(newPostRoutingRules[0]))

				// Check if outdated rule has been removed
				Expect(newPostRoutingRules).ToNot(ContainElement(outdatedRule))
			})
		})
	})

	Describe("EnsureChainsPerCluster", func() {
		BeforeEach(func() {
			err := h.Init()
			Expect(err).To(BeNil())
			tep = forgeValidTEPResource()
		})
		AfterEach(func() {
			err := h.RemoveIPTablesConfigurationPerCluster(tep)
			Expect(err).To(BeNil())
			err = h.Terminate()
			Expect(err).To(BeNil())
		})
		Context("Passing an empty clusterID", func() {
			It("Should return a WrongParameter error", func() {
				err := h.EnsureChainsPerCluster("")
				Expect(err).To(MatchError(fmt.Sprintf("%s must be %s", consts.ClusterIDLabelName, errors.StringNotEmpty)))
			})
		})
		Context("If chains do not exist yet", func() {
			It("Should create chains", func() {
				err := h.EnsureChainsPerCluster(clusterID1)
				Expect(err).To(BeNil())

				natChains, err := ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				filterChains, err := ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				// Check if filter chains have been created by function.
				Expect(filterChains).To(ContainElements(
					strings.Join([]string{liqonetForwardingClusterChainPrefix, strings.Split(clusterID1, "-")[0]}, ""),
					strings.Join([]string{liqonetInputClusterChainPrefix, strings.Split(clusterID1, "-")[0]}, ""),
				))

				// Check if nat chains have been created by function.
				Expect(natChains).To(ContainElements(
					strings.Join([]string{liqonetPostroutingClusterChainPrefix, strings.Split(clusterID1, "-")[0]}, ""),
					strings.Join([]string{liqonetPreroutingClusterChainPrefix, strings.Split(clusterID1, "-")[0]}, ""),
				))
			})
		})
		Context("If chains already exist", func() {
			It("Should be a nop", func() {
				err := h.EnsureChainsPerCluster(clusterID1)
				Expect(err).To(BeNil())

				natChains, err := ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				filterChains, err := ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				// Check if filter chains have been created by function.
				Expect(filterChains).To(ContainElements(
					strings.Join([]string{liqonetForwardingClusterChainPrefix, strings.Split(clusterID1, "-")[0]}, ""),
					strings.Join([]string{liqonetInputClusterChainPrefix, strings.Split(clusterID1, "-")[0]}, ""),
				))

				// Check if nat chains have been created by function.
				Expect(natChains).To(ContainElements(
					strings.Join([]string{liqonetPostroutingClusterChainPrefix, strings.Split(clusterID1, "-")[0]}, ""),
					strings.Join([]string{liqonetPreroutingClusterChainPrefix, strings.Split(clusterID1, "-")[0]}, ""),
				))

				err = h.EnsureChainsPerCluster(clusterID1)
				Expect(err).To(BeNil())

				// Get new chains and assert that they have not changed.
				newNatChains, err := ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				newFilterChains, err := ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				Expect(newNatChains).To(Equal(natChains))
				Expect(newFilterChains).To(Equal(filterChains))
			})
		})
	})

	Describe("Terminate", func() {
		AfterEach(func() {
			err := h.RemoveIPTablesConfigurationPerCluster(tep)
			Expect(err).To(BeNil())
		})
		Context("If there is not a Liqo configuration", func() {
			It("should be a nop", func() {
				err := h.Terminate()
				Expect(err).To(BeNil())
			})
		})
		Context("If there is a Liqo configuration and Liqo chains are not empty", func() {
			It("should remove Liqo configuration", func() {
				err := h.Init()
				Expect(err).To(BeNil())

				// Add a remote cluster config and do not terminate it
				err = h.EnsureChainsPerCluster(clusterID1)
				Expect(err).To(BeNil())
				err = h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(BeNil())

				err = h.Terminate()
				Expect(err).To(BeNil())

				// Check if Liqo chains do exist
				natChains, err := h.ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				filterChains, err := h.ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				Expect(natChains).ToNot(ContainElements(getLiqoChains()))
				Expect(filterChains).ToNot(ContainElements(getLiqoChains()))

				// Check if Liqo rules have been removed

				postRoutingRules, err := h.ListRulesInChain(postroutingChain)
				Expect(err).To(BeNil())
				Expect(postRoutingRules).ToNot(ContainElements([]string{
					fmt.Sprintf("-j %s", liqonetPostroutingChain),
					fmt.Sprintf("-j %s", MASQUERADE),
				}))

				preRoutingRules, err := h.ListRulesInChain(preroutingChain)
				Expect(err).To(BeNil())
				Expect(preRoutingRules).ToNot(ContainElement(fmt.Sprintf("-j %s", liqonetPreroutingChain)))

				forwardRules, err := h.ListRulesInChain(forwardChain)
				Expect(err).To(BeNil())
				Expect(forwardRules).ToNot(ContainElement(fmt.Sprintf("-j %s", liqonetForwardingChain)))

				inputRules, err := h.ListRulesInChain(inputChain)
				Expect(err).To(BeNil())
				Expect(inputRules).ToNot(ContainElement(fmt.Sprintf("-j %s", liqonetInputChain)))
			})
		})
		Context("If there is a Liqo configuration and Liqo chains are empty", func() {
			It("should remove Liqo configuration", func() {
				err := h.Init()
				Expect(err).To(BeNil())

				err = h.Terminate()
				Expect(err).To(BeNil())

				// Check if Liqo chains do exist
				natChains, err := h.ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				filterChains, err := h.ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				Expect(natChains).ToNot(ContainElements(getLiqoChains()))
				Expect(filterChains).ToNot(ContainElements(getLiqoChains()))

				// Check if Liqo rules have been removed

				postRoutingRules, err := h.ListRulesInChain(postroutingChain)
				Expect(err).To(BeNil())
				Expect(postRoutingRules).ToNot(ContainElements([]string{
					fmt.Sprintf("-j %s", liqonetPostroutingChain),
				}))

				preRoutingRules, err := h.ListRulesInChain(preroutingChain)
				Expect(err).To(BeNil())
				Expect(preRoutingRules).ToNot(ContainElement(fmt.Sprintf("-j %s", liqonetPreroutingChain)))

				forwardRules, err := h.ListRulesInChain(forwardChain)
				Expect(err).To(BeNil())
				Expect(forwardRules).ToNot(ContainElement(fmt.Sprintf("-j %s", liqonetForwardingChain)))

				inputRules, err := h.ListRulesInChain(inputChain)
				Expect(err).To(BeNil())
				Expect(inputRules).ToNot(ContainElement(fmt.Sprintf("-j %s", liqonetInputChain)))
			})
		})
	})

	Describe("RemoveIPTablesConfigurationPerCluster", func() {
		BeforeEach(func() {
			err := h.Init()
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			err := h.Terminate()
			Expect(err).To(BeNil())
		})
		Context("If there are no iptables rules/chains related to remote cluster", func() {
			It("should be a nop", func() {
				// Read current configuration in order to compare it
				// with the configuration after func call.
				natChains, err := ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				filterChains, err := ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				postRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)
				Expect(err).To(BeNil())
				forwardRules, err := h.ListRulesInChain(liqonetForwardingChain)
				Expect(err).To(BeNil())
				inputRules, err := h.ListRulesInChain(liqonetInputChain)
				Expect(err).To(BeNil())
				preRoutingRules, err := h.ListRulesInChain(liqonetPreroutingChain)
				Expect(err).To(BeNil())

				err = h.RemoveIPTablesConfigurationPerCluster(tep)
				Expect(err).To(BeNil())

				newNatChains, err := ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				newFilterChains, err := ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				newPostRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)
				Expect(err).To(BeNil())
				newForwardRules, err := h.ListRulesInChain(liqonetForwardingChain)
				Expect(err).To(BeNil())
				newInputRules, err := h.ListRulesInChain(liqonetInputChain)
				Expect(err).To(BeNil())
				newPreRoutingRules, err := h.ListRulesInChain(liqonetPreroutingChain)
				Expect(err).To(BeNil())

				// Assert configs are equal
				Expect(natChains).To(Equal(newNatChains))
				Expect(filterChains).To(Equal(newFilterChains))
				Expect(postRoutingRules).To(Equal(newPostRoutingRules))
				Expect(preRoutingRules).To(Equal(newPreRoutingRules))
				Expect(forwardRules).To(Equal(newForwardRules))
				Expect(inputRules).To(Equal(newInputRules))
			})
		})
		Context("If cluster has an iptables configuration", func() {
			It("should delete chains and rules per cluster", func() {
				// Init configuration
				tep := forgeValidTEPResource()

				err := h.EnsureChainsPerCluster(tep.Spec.ClusterID)
				Expect(err).To(BeNil())

				err = h.EnsureChainRulesPerCluster(tep)
				Expect(err).To(BeNil())

				// Get chains related to cluster
				natChains, err := ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				filterChains, err := ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				// If chain contains clusterID, then it is related to cluster
				natChainsPerCluster := getSliceContainingString(natChains, tep.Spec.ClusterID)
				filterChainsPerCluster := getSliceContainingString(filterChains, tep.Spec.ClusterID)

				// Get rules related to cluster in each chain
				postRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)
				Expect(err).To(BeNil())
				forwardRules, err := h.ListRulesInChain(liqonetForwardingChain)
				Expect(err).To(BeNil())
				inputRules, err := h.ListRulesInChain(liqonetInputChain)
				Expect(err).To(BeNil())
				preRoutingRules, err := h.ListRulesInChain(liqonetPreroutingChain)
				Expect(err).To(BeNil())

				// Rules contains a comment with the clusterID
				clusterPostRoutingRules := getSliceContainingString(postRoutingRules, tep.Spec.ClusterID)
				clusterPreRoutingRules := getSliceContainingString(preRoutingRules, tep.Spec.ClusterID)
				clusterForwardRules := getSliceContainingString(forwardRules, tep.Spec.ClusterID)
				clusterInputRules := getSliceContainingString(inputRules, tep.Spec.ClusterID)

				err = h.RemoveIPTablesConfigurationPerCluster(tep)
				Expect(err).To(BeNil())

				// Read config after call
				newNatChains, err := ipt.ListChains(natTable)
				Expect(err).To(BeNil())
				newFilterChains, err := ipt.ListChains(filterTable)
				Expect(err).To(BeNil())

				newPostRoutingRules, err := h.ListRulesInChain(liqonetPostroutingChain)
				Expect(err).To(BeNil())
				newForwardRules, err := h.ListRulesInChain(liqonetForwardingChain)
				Expect(err).To(BeNil())
				newInputRules, err := h.ListRulesInChain(liqonetInputChain)
				Expect(err).To(BeNil())
				newPreRoutingRules, err := h.ListRulesInChain(liqonetPreroutingChain)
				Expect(err).To(BeNil())

				// Check chains have been removed
				Expect(newNatChains).ToNot(ContainElements(natChainsPerCluster))
				Expect(newFilterChains).ToNot(ContainElements(filterChainsPerCluster))

				// Check rules have been removed
				Expect(newPostRoutingRules).ToNot(ContainElements(clusterPostRoutingRules))
				Expect(newForwardRules).ToNot(ContainElements(clusterForwardRules))
				Expect(newInputRules).ToNot(ContainElements(clusterInputRules))
				Expect(newPreRoutingRules).ToNot(ContainElements(clusterPreRoutingRules))
			})
		})
	})

	Describe("EnsurePostroutingRules", func() {
		BeforeEach(func() {
			err := h.Init()
			Expect(err).To(BeNil())
			err = h.EnsureChainsPerCluster(clusterID1)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			err := h.RemoveIPTablesConfigurationPerCluster(tep)
			Expect(err).To(BeNil())
			err = h.Terminate()
			Expect(err).To(BeNil())
		})
		Context("If tep has an empty clusterID", func() {
			It("should return a WrongParameter error", func() {
				tep.Spec.ClusterID = ""
				err := h.EnsurePostroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", consts.ClusterIDLabelName, errors.StringNotEmpty)))
				tep = forgeValidTEPResource()
			})
		})
		Context("If tep has an invalid LocalPodCIDR", func() {
			It("should return a WrongParameter error", func() {
				tep.Status.LocalPodCIDR = invalidValue
				err := h.EnsurePostroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", localPodCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})
		Context("If tep has an invalid LocalNATPodCIDR", func() {
			It("should return a WrongParameter error", func() {
				tep.Status.LocalNATPodCIDR = invalidValue
				err := h.EnsurePostroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", localNATPodCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})
		Context(fmt.Sprintf("If tep has Status.RemoteNATPodCIDR = %s and an invalid Spec.PodCIDR",
			consts.DefaultCIDRValue), func() {
			It("should return a WrongParameter error", func() {
				tep.Status.RemoteNATPodCIDR = consts.DefaultCIDRValue
				tep.Spec.PodCIDR = invalidValue
				err := h.EnsurePostroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", podCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})
		Context(fmt.Sprintf("If tep has an invalid Status.RemoteNATPodCIDR != %s",
			consts.DefaultCIDRValue), func() {
			It("should return a WrongParameter error", func() {
				tep.Status.RemoteNATPodCIDR = invalidValue
				err := h.EnsurePostroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", remoteNATPodCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})
		Context("Call with different parameters", func() {
			It("should remove old rules and insert updated ones", func() {
				// First call with default tep
				err := h.EnsurePostroutingRules(tep)
				Expect(err).To(BeNil())

				// Get inserted rules
				postRoutingRules, err := h.ListRulesInChain(getClusterPostRoutingChain(clusterID1))
				Expect(err).To(BeNil())

				// Edit tep
				tep.Status.LocalNATPodCIDR = "11.0.0.0/24"

				// Second call
				err = h.EnsurePostroutingRules(tep)
				Expect(err).To(BeNil())

				// Get inserted rules
				newPostRoutingRules, err := h.ListRulesInChain(getClusterPostRoutingChain(clusterID1))
				Expect(err).To(BeNil())

				Expect(newPostRoutingRules).ToNot(ContainElements(postRoutingRules))
				Expect(newPostRoutingRules).To(ContainElements([]string{
					fmt.Sprintf("-s %s -d %s -j %s --to %s", tep.Status.LocalPodCIDR, tep.Status.RemoteNATPodCIDR, NETMAP, tep.Status.LocalNATPodCIDR),
					fmt.Sprintf("! -s %s -d %s -j %s --to-source %s", tep.Status.LocalPodCIDR, tep.Status.RemoteNATPodCIDR, SNAT, mustGetFirstIP(tep.Status.LocalNATPodCIDR)),
				}))
			})
		})
		DescribeTable("EnsurePostroutingRules",
			func(editTep func(), getExpectedRules func() []string) {
				editTep()
				err := h.EnsurePostroutingRules(tep)
				Expect(err).To(BeNil())
				postRoutingRules, err := h.ListRulesInChain(getClusterPostRoutingChain(clusterID1))
				Expect(err).To(BeNil())
				expectedRules := getExpectedRules()
				Expect(postRoutingRules).To(ContainElements(expectedRules))
			},
			Entry(
				fmt.Sprintf("RemoteNATPodCIDR != %s, LocalNATPodCIDR != %s", consts.DefaultCIDRValue, consts.DefaultCIDRValue),
				func() {},
				func() []string {
					return []string{
						fmt.Sprintf("-s %s -d %s -j %s --to %s", tep.Status.LocalPodCIDR, tep.Status.RemoteNATPodCIDR, NETMAP, tep.Status.LocalNATPodCIDR),
						fmt.Sprintf("! -s %s -d %s -j %s --to-source %s", tep.Status.LocalPodCIDR, tep.Status.RemoteNATPodCIDR, SNAT, mustGetFirstIP(tep.Status.LocalNATPodCIDR)),
					}
				},
			),
			Entry(
				fmt.Sprintf("RemoteNATPodCIDR != %s, LocalNATPodCIDR = %s", consts.DefaultCIDRValue, consts.DefaultCIDRValue),
				func() { tep.Status.LocalNATPodCIDR = consts.DefaultCIDRValue },
				func() []string {
					return []string{fmt.Sprintf("! -s %s -d %s -j %s --to-source %s", tep.Status.LocalPodCIDR, tep.Status.RemoteNATPodCIDR, SNAT, mustGetFirstIP(tep.Status.LocalPodCIDR))}
				},
			),
			Entry(
				fmt.Sprintf("RemoteNATPodCIDR = %s, LocalNATPodCIDR != %s", consts.DefaultCIDRValue, consts.DefaultCIDRValue),
				func() { tep.Status.RemoteNATPodCIDR = consts.DefaultCIDRValue },
				func() []string {
					return []string{
						fmt.Sprintf("-s %s -d %s -j %s --to %s", tep.Status.LocalPodCIDR, tep.Spec.PodCIDR, NETMAP, tep.Status.LocalNATPodCIDR),
						fmt.Sprintf("! -s %s -d %s -j %s --to-source %s", tep.Status.LocalPodCIDR, tep.Spec.PodCIDR, SNAT, mustGetFirstIP(tep.Status.LocalNATPodCIDR)),
					}
				},
			),
			Entry(fmt.Sprintf("RemoteNATPodCIDR = %s, LocalNATPodCIDR = %s", consts.DefaultCIDRValue, consts.DefaultCIDRValue),
				func() {
					tep.Status.RemoteNATPodCIDR = consts.DefaultCIDRValue
					tep.Status.LocalNATPodCIDR = consts.DefaultCIDRValue
				},
				func() []string {
					return []string{fmt.Sprintf("! -s %s -d %s -j %s --to-source %s", tep.Status.LocalPodCIDR, tep.Spec.PodCIDR, SNAT, mustGetFirstIP(tep.Status.LocalPodCIDR))}
				},
			),
		)
	})
	Describe("EnsurePreroutingRules", func() {
		BeforeEach(func() {
			err := h.Init()
			Expect(err).To(BeNil())
			err = h.EnsureChainsPerCluster(clusterID1)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			err := h.RemoveIPTablesConfigurationPerCluster(tep)
			Expect(err).To(BeNil())
			err = h.Terminate()
			Expect(err).To(BeNil())
		})
		Context("If tep has an empty clusterID", func() {
			It("should return a WrongParameter error", func() {
				tep.Spec.ClusterID = ""
				err := h.EnsurePreroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", consts.ClusterIDLabelName, errors.StringNotEmpty)))
				tep = forgeValidTEPResource()
			})
		})
		Context("If tep has an invalid LocalPodCIDR", func() {
			It("should return a WrongParameter error", func() {
				tep.Status.LocalPodCIDR = invalidValue
				err := h.EnsurePreroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", localPodCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})
		Context("If tep has an invalid LocalNATPodCIDR", func() {
			It("should return a WrongParameter error", func() {
				tep.Status.LocalNATPodCIDR = invalidValue
				err := h.EnsurePreroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", localNATPodCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})
		Context(fmt.Sprintf("If tep has Status.RemoteNATPodCIDR = %s and an invalid Spec.PodCIDR",
			consts.DefaultCIDRValue), func() {
			It("should return a WrongParameter error", func() {
				tep.Status.RemoteNATPodCIDR = consts.DefaultCIDRValue
				tep.Spec.PodCIDR = invalidValue
				err := h.EnsurePreroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", podCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})
		Context(fmt.Sprintf("If tep has an invalid Status.RemoteNATPodCIDR != %s",
			consts.DefaultCIDRValue), func() {
			It("should return a WrongParameter error", func() {
				tep.Status.RemoteNATPodCIDR = invalidValue
				err := h.EnsurePreroutingRules(tep)
				Expect(err).To(MatchError(fmt.Sprintf("invalid TunnelEndpoint resource: %s must be %s", remoteNATPodCIDR, errors.ValidCIDR)))
				tep = forgeValidTEPResource()
			})
		})
		Context("Call with different parameters", func() {
			It("should remove old rules and insert updated ones", func() {
				// First call with default tep
				err := h.EnsurePreroutingRules(tep)
				Expect(err).To(BeNil())

				// Get inserted rules
				preRoutingRules, err := h.ListRulesInChain(getClusterPreRoutingChain(clusterID1))
				Expect(err).To(BeNil())

				// Edit tep
				tep.Status.LocalNATPodCIDR = "11.0.0.0/24"

				// Second call
				err = h.EnsurePreroutingRules(tep)
				Expect(err).To(BeNil())

				// Get inserted rules
				newPreRoutingRules, err := h.ListRulesInChain(getClusterPreRoutingChain(clusterID1))
				Expect(err).To(BeNil())

				Expect(newPreRoutingRules).ToNot(ContainElements(preRoutingRules))
				Expect(newPreRoutingRules).To(ContainElement(fmt.Sprintf("-s %s -d %s -j %s --to %s",
					tep.Status.RemoteNATPodCIDR, tep.Status.LocalNATPodCIDR, NETMAP, tep.Status.LocalPodCIDR)))
			})
		})
		DescribeTable("EnsurePreroutingRules",
			func(editTep func(), getExpectedRules func() []string) {
				editTep()
				err := h.EnsurePreroutingRules(tep)
				Expect(err).To(BeNil())
				preRoutingRules, err := h.ListRulesInChain(getClusterPreRoutingChain(clusterID1))
				Expect(err).To(BeNil())
				expectedRules := getExpectedRules()
				Expect(preRoutingRules).To(ContainElements(expectedRules))
			},
			Entry(
				fmt.Sprintf("LocalNATPodCIDR != %s", consts.DefaultCIDRValue),
				func() {},
				func() []string {
					return []string{fmt.Sprintf("-s %s -d %s -j %s --to %s",
						tep.Status.RemoteNATPodCIDR, tep.Status.LocalNATPodCIDR, NETMAP, tep.Status.LocalPodCIDR)}
				},
			),
			Entry(
				fmt.Sprintf("LocalNATPodCIDR = %s", consts.DefaultCIDRValue),
				func() { tep.Status.LocalNATPodCIDR = consts.DefaultCIDRValue },
				func() []string { return []string{} },
			),
		)
	})
})

func forgeValidTEPResource() *v1alpha1.TunnelEndpoint {
	return &v1alpha1.TunnelEndpoint{
		Spec: v1alpha1.TunnelEndpointSpec{
			ClusterID:     clusterID1,
			PodCIDR:       "10.0.0.0/24",
			ExternalCIDR:  "10.0.1.0/24",
			EndpointIP:    "172.10.0.2",
			BackendType:   "backendType",
			BackendConfig: make(map[string]string),
		},
		Status: v1alpha1.TunnelEndpointStatus{
			LocalPodCIDR:          "192.168.0.0/24",
			LocalNATPodCIDR:       "192.168.1.0/24",
			RemoteNATPodCIDR:      remoteNATPodCIDRValue1,
			LocalExternalCIDR:     "192.168.3.0/24",
			LocalNATExternalCIDR:  "192.168.4.0/24",
			RemoteNATExternalCIDR: "192.168.5.0/24",
		},
	}
}

func mustGetFirstIP(network string) string {
	firstIP, err := utils.GetFirstIP(network)
	if err != nil {
		klog.Error(err)
		os.Exit(1)
	}
	return firstIP
}
