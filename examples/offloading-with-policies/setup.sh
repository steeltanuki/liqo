#!/bin/bash

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

CLUSTER_NAME_1=venice
CLUSTER_NAME_2=florence
CLUSTER_NAME_3=naples

KUBECONFIG_1=liqo_kubeconf_venice
KUBECONFIG_2=liqo_kubeconf_florence
KUBECONFIG_3=liqo_kubeconf_naples

rm -f "$KUBECONFIG_1" "$KUBECONFIG_2" "$KUBECONFIG_3"

check_requirements $CLUSTER_COMMAND

delete_clusters $CLUSTER_COMMAND "$CLUSTER_NAME_1" "$CLUSTER_NAME_2" "$CLUSTER_NAME_3"

create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_1" "$KUBECONFIG_1" "$LIQO_CLUSTER_CONFIG_YAML"
create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_2" "$KUBECONFIG_2" "$LIQO_CLUSTER_CONFIG_YAML"
create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_3" "$KUBECONFIG_3" "$LIQO_CLUSTER_CONFIG_YAML"


if [ "$CLUSTER_COMMAND" = "k3d" ]; then
    install_liqo_k3d "$CLUSTER_NAME_1" "$KUBECONFIG_1" "10.42.0.0/16" "10.43.0.0/16" "topology.liqo.io/region=north"
    install_liqo_k3d "$CLUSTER_NAME_2" "$KUBECONFIG_2" "10.42.0.0/16" "10.43.0.0/16" "topology.liqo.io/region=center"
    install_liqo_k3d "$CLUSTER_NAME_3" "$KUBECONFIG_3" "10.42.0.0/16" "10.43.0.0/16" "topology.liqo.io/region=south"
else
    install_liqo $CLUSTER_COMMAND "$CLUSTER_NAME_1" "$KUBECONFIG_1" "topology.liqo.io/region=north"
    install_liqo $CLUSTER_COMMAND "$CLUSTER_NAME_2" "$KUBECONFIG_2" "topology.liqo.io/region=center"
    install_liqo $CLUSTER_COMMAND "$CLUSTER_NAME_3" "$KUBECONFIG_3" "topology.liqo.io/region=south"
fi



