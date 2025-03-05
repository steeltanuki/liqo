#!/bin/bash

set -e           # Fail in case of error
set -o nounset   # Fail if undefined variables are used
set -o pipefail  # Fail if one of the piped commands fails
set -x

function setup_colors() {
    # Only use colors if connected to a terminal
    if [ -t 1 ]; then
        RED=$(printf '\033[31m')
        GREEN=$(printf '\033[32m')
        YELLOW=$(printf '\033[33m')
        BLUE=$(printf '\033[34m')
        BOLD=$(printf '\033[1m')
        RESET=$(printf '\033[m')
        PREVIOUS_LINE=$(printf '\e[1A')
        CLEAR_LINE=$(printf '\e[K')
    else
        RED=""
        GREEN=""
        YELLOW=""
        BLUE=""
        BOLD=""
        RESET=""
        PREVIOUS_LINE=""
        CLEAR_LINE=""
    fi
}

function error() {
    echo -e "${RED}${BOLD}ERROR${RESET}\t$1"
}

function warning() {
    echo -e "${YELLOW}${BOLD}WARN${RESET}\t$1"
}

function info() {
    echo -e "${BLUE}${BOLD}INFO${RESET}\t$1"
}

function success_clear_line() {
    echo -e "${PREVIOUS_LINE}${CLEAR_LINE}${GREEN}${BOLD}SUCCESS${RESET}\t$1"
}

function success() {
    echo -e "${GREEN}${BOLD}SUCCESS${RESET}\t$1"
}

function check_cluster_command() {
  if [ "$1" != "kind" ] && [ "$1" != "k3d" ]; then
        error "Unknown cluster command type (must be one of 'kind' or 'k3d'): $1"
        exit 1
    fi
}


function check_requirements() {

    local cluster_command="${1:-kind}"
    local cluster_command_reference="https://kind.sigs.k8s.io/docs/user/quick-start/#installation"

    check_cluster_command "$cluster_command"

    if [ "$cluster_command" = "k3d" ]; then
        cluster_command_reference="https://k3d.io/stable/#installation"
    fi

    if ! command -v docker &> /dev/null;
    then
        error "Docker engine could not be found on your system. Please install docker engine to continue: https://docs.docker.com/get-docker/"
        exit 1
    fi

    if ! docker info &> /dev/null;
    then
        error "Docker is not running. Please start it to continue."
        exit 1
    fi

    if ! command -v kubectl &> /dev/null;
    then
        error "Kubectl could not be found on your system. Please install kubectl to continue: https://kubernetes.io/docs/tasks/tools/#kubectl"
        exit 1
    fi

    if ! command -v $cluster_command &> /dev/null;
    then
        error "$cluster_command could not be found on your system. Please install $cluster_command to continue: $cluster_command_reference"
        exit 1
    fi

    if ! command -v liqoctl &> /dev/null;
    then
        error "Liqoctl could not be found on your system. Please install liqoctl to continue"
        exit 1
    fi

    # check for extra requirements
    for cmd in "$@"; do
        if ! command -v "$cmd" &> /dev/null;
        then
            error "Command $cmd could not be found on your system. Please install it to continue."
            exit 1
        fi
    done
}

function delete_clusters() {  
    
    if [ "$1" = "k3d" ]; then
        delete_k3d_clusters ${@:2}
    elif [ "$1" = "kind" ]; then
        delete_kind_clusters ${@:2}
    else
        delete_kind_clusters ${@}
    fi
}


function delete_kind_clusters() {
    for cluster in "$@"; do
        info "Ensuring that no cluster \"$cluster\" is running..."
        kind delete cluster --name "$cluster" > /dev/null 2>&1
        success_clear_line "No cluster \"${cluster}\" is running."
    done
}

function create_cluster() {
    if [ "$1" = "k3d" ]; then
        create_k3d_cluster ${@:2}
    elif [ "$1" = "kind" ]; then
        create_kind_cluster ${@:2}
    else
        create_kind_cluster ${@}
    fi
}

function create_kind_cluster() {
    local name="$1"
    local kubeconfig="$2"
    local config="$3"

    info "Creating cluster \"$name\"..."
    fail_on_error "kind create cluster --name $name \
        --kubeconfig $kubeconfig --config $config --wait 5m" "Failed to create cluster \"$name\""
    success_clear_line "Cluster \"$name\" has been created."
}

function install_liqo() {
    local cluster_name="$1"
    local kubeconfig="$2"

    info "Installing liqo on cluster \"$cluster_name\"..."

    shift 2
    labels="$*"

    fail_on_error "liqoctl install kind --cluster-id $cluster_name \
        --cluster-labels=$(join_by , "${labels[@]}") \
        --kubeconfig $kubeconfig" "Failed to install liqo on cluster \"$cluster_name\""

    success_clear_line "Liqo has been installed on cluster \"$cluster_name\"."
}

function install_liqo_k3d() {
    local cluster_name="$1"
    local kubeconfig="$2"
    local pod_cidr="$3"
    local service_cidr="$4"

    if [ -z "$pod_cidr" ]; then
        pod_cidr="10.42.0.0/16"
    fi
    if [ -z "$service_cidr" ]; then
        service_cidr="10.43.0.0/16"
    fi

    info "Installing liqo on cluster \"$cluster_name\"..."

    shift 4
    labels="$*"

    api_server_address=$(kubectl get nodes --kubeconfig "$kubeconfig" --selector=node-role.kubernetes.io/master -o jsonpath='{$.items[*].status.addresses[?(@.type=="InternalIP")].address}')

    fail_on_error "liqoctl install k3s --cluster-id $cluster_name \
        --cluster-labels=$(join_by , "${labels[@]}") \
        --pod-cidr $pod_cidr \
        --service-cidr $service_cidr \
        --api-server-url https://$api_server_address:6443 \
        --kubeconfig $kubeconfig" "Failed to install liqo on cluster \"${cluster_name}\""

    success_clear_line "Liqo has been installed on cluster \"$cluster_name\"."
}

function delete_k3d_clusters() {
    for cluster in "$@"; do
        info "Ensuring that no cluster \"$cluster\" is running..."
        k3d cluster delete "$cluster" > /dev/null 2>&1
        success_clear_line "No cluster \"${cluster}\" is running."
    done
}

function create_k3d_cluster() {
    if [ $# -eq 2 ]; then
        local name="$1"
        local config="$2"
    elif [ $# -eq 3 ]; then
        local name="$1"
        local kubeconfig="$2"
        local config="$3"
    else
        error "Invalid number of arguments for create_k3d_cluster"
        exit 1
    fi

    info "Creating cluster \"$name\"..."
    fail_on_error "k3d cluster create $name -c $config --kubeconfig-update-default=false" "Failure to create cluster \"${name}\""
    if [ -n "$kubeconfig" ]; then
        k3d kubeconfig write $name --overwrite -o $kubeconfig  
    fi
    success_clear_line "Cluster \"$name\" has been created."
}

function get_k3d_kubeconfig() {
    local name="$1"

    k3d kubeconfig write "$name"
}

function install_k8gb() {
    local kubeconfig="$1"
    local cluster_geo_tag="$2"
    local cluster_ext_geo_tag="$3"
    local dns_ip="$4"

    info "Installing k8gb on cluster..."

    fail_on_error "kubectl create namespace k8gb --kubeconfig $kubeconfig" "Failed to create namespace k8gb"
    fail_on_error "kubectl -n k8gb create secret generic rfc2136 --kubeconfig $kubeconfig --from-literal=secret=96Ah/a2g0/nLeFGK+d/0tzQcccf9hCEIy34PoXX2Qg8=" "Failed to create secret"

    fail_on_error "helm -n k8gb upgrade -i k8gb k8gb/k8gb --kubeconfig $kubeconfig \
        --set k8gb.clusterGeoTag=$cluster_geo_tag \
        --set k8gb.extGslbClustersGeoTags=$cluster_ext_geo_tag \
        --set k8gb.reconcileRequeueSeconds=10 \
        --set k8gb.dnsZoneNegTTL=10 \
        --set k8gb.imageTag=v0.9.0 \
        --set k8gb.log.format=simple \
        --set k8gb.log.level=debug \
        --set rfc2136.enabled=true \
        --set k8gb.edgeDNSServers[0]=${dns_ip}:30053 \
        --set externaldns.image=absaoss/external-dns:rfc-ns1 \
        --wait --timeout=2m0s" "Failed to install k8gb"

    success_clear_line "K8gb has been installed on cluster."
}

function install_ingress_nginx() {
    local kubeconfig="$1"
    local namespace="$2"
    local values="$3"
    local version="$4"

    if [ -z "$version" ]; then
        version="4.0.15"
    fi

    info "Installing ingress-nginx on cluster..."

    fail_on_error "helm -n $namespace upgrade --kubeconfig $kubeconfig -i nginx-ingress nginx-stable/ingress-nginx \
	    --version $version -f $values" "Failed to install ingress-nginx"

    success_clear_line "Ingress-nginx has been installed on cluster."
}

function fail_on_error() {
    local cmd="$1"
    local msg="$2"

    set +e
    output=$($cmd 2>&1)
    # shellcheck disable=SC2181
    # we need to collect the output and then check the exit code
    if [ $? -ne 0 ]; then
        error "$msg: ${output}"
        exit 1
    fi
    set -e
}

function join_by() {
    local IFS="$1"
    shift
    echo "$*"
}

setup_colors
