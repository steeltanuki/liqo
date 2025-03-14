#!/usr/bin/env bash

set -e

here="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
# shellcheck source=/dev/null
source "$here/../common.sh"

CLUSTER_COMMAND="${1:-kind}"

if [ "$CLUSTER_COMMAND" != "kind" ] && [ "$CLUSTER_COMMAND" != "k3d" ]; then
    echo "Unknown cluster type: $CLUSTER_COMMAND"
    exit 1
fi

if [ "$CLUSTER_COMMAND" = "k3d" ]; then
    LIQO_CLUSTER_CONFIG_YAML="$here/manifests/cluster-k3d.yaml"  
else
    LIQO_CLUSTER_CONFIG_YAML="$here/manifests/cluster.yaml"
fi

CLUSTER_NAME_ORIGIN=europe-cloud
CLUSTER_NAME_DESTINATION1=europe-rome-edge
CLUSTER_NAME_DESTINATION2=europe-milan-edge

CLUSTER_LABEL_ORIGIN="topology.liqo.io/type=origin"
CLUSTER_LABEL_DESTINATION="topology.liqo.io/type=destination"

KUBECONFIG_ORIGIN=liqo_kubeconf_europe-cloud
KUBECONFIG_DESTINATION1=liqo_kubeconf_europe-rome-edge
KUBECONFIG_DESTINATION2=liqo_kubeconf_europe-milan-edge

rm -f "$KUBECONFIG_ORIGIN" "$KUBECONFIG_DESTINATION1" "$KUBECONFIG_DESTINATION2"

check_requirements $CLUSTER_COMMAND

delete_clusters $CLUSTER_COMMAND "$CLUSTER_NAME_ORIGIN" "$CLUSTER_NAME_DESTINATION1" "$CLUSTER_NAME_DESTINATION2"

create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_ORIGIN" "$KUBECONFIG_ORIGIN" "$LIQO_CLUSTER_CONFIG_YAML"
create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_DESTINATION1" "$KUBECONFIG_DESTINATION1" "$LIQO_CLUSTER_CONFIG_YAML"
create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_DESTINATION2" "$KUBECONFIG_DESTINATION2" "$LIQO_CLUSTER_CONFIG_YAML"


if [ "$CLUSTER_COMMAND" = "k3d" ]; then
    install_liqo_k3d "$CLUSTER_NAME_ORIGIN" "$KUBECONFIG_ORIGIN" "10.42.0.0/16" "10.43.0.0/16" "$CLUSTER_LABEL_ORIGIN"
    install_liqo_k3d "$CLUSTER_NAME_DESTINATION1" "$KUBECONFIG_DESTINATION1" "10.42.0.0/16" "10.43.0.0/16" "$CLUSTER_LABEL_DESTINATION"
    install_liqo_k3d "$CLUSTER_NAME_DESTINATION2" "$KUBECONFIG_DESTINATION2" "10.42.0.0/16" "10.43.0.0/16" "$CLUSTER_LABEL_DESTINATION"
else
    install_liqo "$CLUSTER_NAME_ORIGIN" "$KUBECONFIG_ORIGIN" "$CLUSTER_LABEL_ORIGIN"
    install_liqo "$CLUSTER_NAME_DESTINATION1" "$KUBECONFIG_DESTINATION1" "$CLUSTER_LABEL_DESTINATION"
    install_liqo "$CLUSTER_NAME_DESTINATION2" "$KUBECONFIG_DESTINATION2" "$CLUSTER_LABEL_DESTINATION"
fi


