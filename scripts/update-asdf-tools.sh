#!/bin/sh

# shellcheck disable=SC2016

set -e
set -o errexit -o nounset

cut -d ' ' -f 1 < .tool-versions | \
xargs -I {} -n 1 sh -c 'echo {} $(asdf latest {} 2>/dev/null)' | \
sed 's/^jq$/jq 1.6/g' > .tool-versions.tmp

mv .tool-versions.tmp .tool-versions
