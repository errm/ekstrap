#!/bin/bash

set -euo pipefail

openssl aes-256-cbc -K $encrypted_189aefeda93d_key -iv $encrypted_189aefeda93d_iv -in .travis/ekstrap.asc.enc -out .travis/ekstrap.asc -d
gpg --import .travis/ekstrap.asc
make install-linter
