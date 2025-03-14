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
    LIQO_CLUSTER_CONFIG1_YAML="$here/manifests/cluster1-k3d.yaml"
    LIQO_CLUSTER_CONFIG2_YAML="$here/manifests/cluster2-k3d.yaml"  
else
    LIQO_CLUSTER_CONFIG1_YAML="$here/manifests/cluster1.yaml"
    LIQO_CLUSTER_CONFIG2_YAML="$here/manifests/cluster2.yaml"
fi

CLUSTER_NAME_1=turin
CLUSTER_NAME_2=lyon

KUBECONFIG_1=liqo_kubeconf_turin
KUBECONFIG_2=liqo_kubeconf_lyon

rm -f "$KUBECONFIG_1" "$KUBECONFIG_2"

check_requirements $CLUSTER_COMMAND

delete_clusters $CLUSTER_COMMAND "$CLUSTER_NAME_1" "$CLUSTER_NAME_2"

create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_1" "$KUBECONFIG_1" "$LIQO_CLUSTER_CONFIG1_YAML"
create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_2" "$KUBECONFIG_2" "$LIQO_CLUSTER_CONFIG2_YAML"

if [ "$CLUSTER_COMMAND" = "k3d" ]; then
    install_liqo_k3d "$CLUSTER_NAME_1" "$KUBECONFIG_1" "10.42.0.0/16" "10.43.0.0/16"
    install_liqo_k3d "$CLUSTER_NAME_2" "$KUBECONFIG_2" "10.44.0.0/16" "10.45.0.0/16"
else
    install_liqo "$CLUSTER_NAME_1" "$KUBECONFIG_1"
    install_liqo "$CLUSTER_NAME_2" "$KUBECONFIG_2"
fi
