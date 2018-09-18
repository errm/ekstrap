#!/bin/bash

set -euo pipefail

strip -x ./dist/linux_amd64/ekstrap
upx -9 ./dist/linux_amd64/ekstrap
