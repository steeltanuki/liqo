#!/bin/bash

set -e
set -x

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

CLUSTER_NAME_1=rome
CLUSTER_NAME_2=milan

KUBECONFIG_1=liqo_kubeconf_rome
KUBECONFIG_2=liqo_kubeconf_milan

rm -f "$KUBECONFIG_1" "$KUBECONFIG_2"

check_requirements $CLUSTER_COMMAND

delete_clusters $CLUSTER_COMMAND "$CLUSTER_NAME_1" "$CLUSTER_NAME_2"

create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_1" "$KUBECONFIG_1" "$LIQO_CLUSTER_CONFIG_YAML"
create_cluster $CLUSTER_COMMAND "$CLUSTER_NAME_2" "$KUBECONFIG_2" "$LIQO_CLUSTER_CONFIG_YAML"
