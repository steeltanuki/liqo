package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo/pkg/liqoctl/install"
	installutils "github.com/liqotech/liqo/pkg/liqoctl/install/utils"
)

// installCmd represents the generateInstall command.
func newInstallCommand(ctx context.Context) *cobra.Command {
	var installCmd = &cobra.Command{
		Use:   installutils.LiqoctlInstallCommand,
		Short: installutils.LiqoctlInstallShortHelp,
		Long:  installutils.LiqoctlInstallLongHelp,
		Run: func(cmd *cobra.Command, args []string) {
			install.HandleInstallCommand(ctx, cmd, os.Args[0])
		},
	}

	installCmd.Flags().StringP("provider", "p", "kubeadm", "Select the cluster provider type")
	installCmd.Flags().IntP("timeout", "t", 600, "Configure the timeout for the installation process in seconds")
	installCmd.Flags().StringP("version", "", "", "Select the Liqo version (default: latest stable release)")
	installCmd.Flags().BoolP("devel", "", false,
		"Enable use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored")
	installCmd.Flags().BoolP("only-output-values", "", false, "Generate a values file for further customization")
	installCmd.Flags().StringP("dump-values-path", "", "./values.yaml", "Path for the output value file")
	installCmd.Flags().BoolP("dry-run", "", false, "Simulate an install")
	installCmd.Flags().BoolP("enable-lan-discovery", "", true, "Enable LAN discovery")
	installCmd.Flags().StringP("cluster-labels", "", "",
		"Cluster Labels to append to Liqo Cluster, supports '='.(e.g. --cluster-labels key1=value1,key2=value2)")
	installCmd.Flags().BoolP("disable-endpoint-check", "", false,
		"Disable the check that the current kubeconfig context contains the same endpoint retrieved from the cloud provider (AKS, EKS, GKE)")
	installCmd.Flags().String("chart-path", installutils.LiqoChartFullName,
		"Specify a path to get the Liqo chart, instead of installing the chart from the official repository")

	for _, p := range providers {
		initFunc, ok := providerInitFunc[p]
		if !ok {
			klog.Fatalf("unknown provider: %v", p)
		}
		initFunc(installCmd.Flags())
	}
	return installCmd
}
