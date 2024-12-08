#!/usr/bin/env bash

set -e

git_tag=$(git describe --exact-match)
echo curl -H"Authorization: token $SRHT_TOKEN" \
    https://git.sr.ht/api/~hedy/repos/gelim/artifacts/$git_tag \
    -F "file=@$1"
