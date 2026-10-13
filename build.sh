#!/bin/bash
#
# Copyright 2023 Enflame. All Rights Reserved.
#

if [ -f "../../g_version.txt" ]; then
    source ../../g_version.txt
fi
PKG_VER=${GCU_FEATURE_DISCOVERY_VERSION:-"0.0.0"}

function usage() {
  cat <<EOF
Usage: ./build.sh [command]
Command:
    all          Build the gcu-feature-discovery
    clean        Clean the gcu-feature-discovery
Example:
    ./build.sh all
EOF
}

function build_all() {
    # clean old dist if exist
    if [ -d "./dist" ]; then
        rm -rf ./dist
    fi
    echo -e "\033[33mBuilding gcu-feature-discovery package start...\033[0m"

    PKG_DIR="dist/gcu-feature-discovery_${PKG_VER}"
    mkdir -p $PKG_DIR
    make

    cp -rf -L deployments/* ${PKG_DIR}/
    mv ./gcu-feature-discovery ${PKG_DIR}/

    cp -rf config ${PKG_DIR}/
    cp -rf topscloud.json ${PKG_DIR}/config/

    # The all-in-one deployment assets are optional and may be absent.
    if [ -d "all-in-one" ]; then
        cp -rf all-in-one/show-labels.sh ${PKG_DIR}/
        cp -rf -L all-in-one ${PKG_DIR}/
        cp -rf topscloud.json ${PKG_DIR}/all-in-one/config/
    fi

    echo -e "\033[33mBuilding gcu-feature-discovery package successfully!\033[0m"
}

function clean_all() {
    if [ -d "./dist" ]; then
        rm -rf ./dist
    fi
    echo -e "\033[33mClean gcu-feature-discovery package successfully.\033[0m"
}

function main() {
    [[ "$EUID" -ne 0 ]] && { echo "[INFO] Run as non-root"; }
    [[ "$#" -lt 1 ]] && { usage >&2; exit 1; }

    case $1 in
    "clean")
        clean_all;;
    "all")
        build_all;;
    *)
        usage
        exit 1;;
    esac
}

main "$@"
