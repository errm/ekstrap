#!/bin/bash

set -euo pipefail

GORELEASER_VERSION="0.79.0"
GORELEASER_CHECKSUM="8b3f62f582bddde2a29b1d33baaf92b66281312c2fd8a3d38ad2bac8e35c14dd"

export DEBIAN_FRONTEND=noninteractive

docker login -u=errm -p="$DOCKER_PASSWORD"

sudo -E apt-get -yq update
sudo -E apt-get -yq \
  --no-install-suggests \
  --no-install-recommends \
  --force-yes install rpm upx
gem install package_cloud

curl -LO "https://github.com/goreleaser/goreleaser/releases/download/v$GORELEASER_VERSION/goreleaser_amd64.deb"
echo "echo $GORELEASER_CHECKSUM goreleaser_amd64.deb" | sha256sum --check --status -
sudo apt install ./goreleaser_amd64.deb

make release

DEBS="ubuntu/xenial ubuntu/bionic debian/jessie debian/stretch debian/buster"
HATS="el/6 el/7 fedora/27 fedora/28"

for DISTRO in $DEBS
do
  package_cloud push errm/ekstrap/$DISTRO dist/ekstrap.deb
done

for DISTRO in $HATS
do
  package_cloud push errm/ekstrap/$DISTRO dist/ekstrap.rpm
done
