#!/usr/bin/env bash

set -e

git_tag=$(git describe --exact-match)
echo $git_tag
ls dist/gelim*{.txt,.tar.gz} | xargs $1 -I % \
    curl -H"Authorization: token $SRHT_TOKEN" \
    https://git.sr.ht/api/~hedy/repos/gelim/artifacts/$git_tag \
    -F "file=@%"
