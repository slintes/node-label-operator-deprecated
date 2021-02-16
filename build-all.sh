#!/bin/bash

set -e

export OPERATOR_IMG=quay.io/slintes/node-label-operator:latest
export BUNDLE_IMG=quay.io/slintes/node-label-operator-bundle:latest

make docker-build docker-push IMG=$OPERATOR_IMG
make bundle IMG=$OPERATOR_IMG
make bundle-build BUNDLE_IMG=$BUNDLE_IMG
make docker-push IMG=$BUNDLE_IMG

