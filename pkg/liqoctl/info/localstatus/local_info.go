// Copyright 2019-2024 The Liqo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package localstatus

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	liqov1beta1 "github.com/liqotech/liqo/apis/core/v1beta1"
	"github.com/liqotech/liqo/pkg/liqoctl/info"
	"github.com/liqotech/liqo/pkg/liqoctl/output"
	liqoctlutils "github.com/liqotech/liqo/pkg/liqoctl/utils"
	"github.com/liqotech/liqo/pkg/utils"
	"github.com/liqotech/liqo/pkg/utils/apiserver"
	"github.com/liqotech/liqo/pkg/utils/getters"
)

// Installation contains the info about the local Liqo installation.
type Installation struct {
	ClusterID     liqov1beta1.ClusterID `json:"clusterID"`
	Version       string                `json:"version"`
	Labels        map[string]string     `json:"labels"`
	APIServerAddr string
}

// InstallationChecker collects the info about the local Liqo installation.
type InstallationChecker struct {
	info.CheckerCommon
	data Installation
}

const (
	ctrlManagerContainerName = "controller-manager"
)

// Collect data about the local installation of Liqo.
func (l *InstallationChecker) Collect(ctx context.Context, options info.Options) {
	// Get the cluster ID of the local cluster
	clusterID, err := utils.GetClusterID(ctx, options.KubeClient, options.LiqoNamespace)
	if err != nil {
		l.AddCollectionError(fmt.Errorf("unable to get cluster id: %w", err))
	}
	l.data.ClusterID = clusterID

	// Collect Liqo version and cluster labels from the controller manager deployment
	ctrlDeployment, err := getters.GetControllerManagerDeployment(ctx, options.CRClient, options.LiqoNamespace)
	if err != nil {
		l.AddCollectionError(fmt.Errorf("unable to get Liqo version and cluster labels: %w", err))
	} else {
		if err := l.collectLiqoVersion(ctrlDeployment); err != nil {
			l.AddCollectionError(fmt.Errorf("unable to get Liqo version: %w", err))
		}

		if err := l.collectClusterLabels(ctrlDeployment); err != nil {
			l.AddCollectionError(fmt.Errorf("unable to get cluster labels: %w", err))
		}
	}

	// Get the URL of the K8s API
	apiAddr, err := apiserver.GetURL(ctx, options.CRClient, "")
	if err != nil {
		l.AddCollectionError(fmt.Errorf("unable to get K8s API server: %w", err))
	}
	l.data.APIServerAddr = apiAddr
}

// Format returns the collected data using a user friendly output.
func (l *InstallationChecker) Format(options info.Options) string {
	main := output.NewRootSection()
	main.AddEntry("Cluster ID", string(l.data.ClusterID))
	main.AddEntry("Version", l.data.Version)
	main.AddEntry("K8s API server", l.data.APIServerAddr)
	labelsSection := main.AddSection("Cluster labels")
	for key, val := range l.data.Labels {
		labelsSection.AddEntry(key, val)
	}
	return main.SprintForBox(options.Printer)
}

// GetData returns the data collected by the checker.
func (l *InstallationChecker) GetData() interface{} {
	return l.data
}

// GetID returns the id of the section collected by the checker.
func (l *InstallationChecker) GetID() string {
	return "local"
}

// GetTitle returns the title of the section collected by the checker.
func (l *InstallationChecker) GetTitle() string {
	return "Local installation info"
}

func (l *InstallationChecker) collectClusterLabels(ctrlDeployment *appsv1.Deployment) error {
	var ctrlContainer corev1.Container

	// Get the container of the controller manager
	containers := ctrlDeployment.Spec.Template.Spec.Containers
	for i := range containers {
		if containers[i].Name == ctrlManagerContainerName {
			ctrlContainer = containers[i]
		}
	}

	clusterLabelsArg, err := liqoctlutils.ExtractValuesFromArgumentList("--cluster-labels", ctrlContainer.Args)
	if err != nil {
		return err
	}

	clusterLabels, err := liqoctlutils.ParseArgsMultipleValues(clusterLabelsArg, ",")
	if err != nil {
		return err
	}

	l.data.Labels = clusterLabels
	return nil
}

func (l *InstallationChecker) collectLiqoVersion(ctrlDeployment *appsv1.Deployment) error {
	version, err := getters.GetContainerImageVersion(ctrlDeployment.Spec.Template.Spec.Containers, ctrlManagerContainerName)
	if err != nil {
		return err
	}
	l.data.Version = version
	return nil
}
