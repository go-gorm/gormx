#!/usr/bin/env bash

set -e

installed_deps=$(ls `go env GOPATH`/bin/)

deps=(
  "goimports;golang.org/x/tools/cmd/goimports" # go 官方的 goimports 工具
  "gofumpt;mvdan.cc/gofumpt" # 格式化工具
)

for i in "${deps[@]}"; do
  dep_name=$(echo "$i" | cut -d ';' -f 1-1)
  dep_path=$(echo "$i" | cut -d ';' -f 2-2)
  if echo $installed_deps | grep -w "$dep_name" > /dev/null; then
    printf $(tput setaf 2)"\"$dep_name\""$(tput sgr0)" installed, skip.\n"
  else
    echo "go install $dep_path@latest" && go install "$dep_path"@latest;
  fi
done

printf "\n"

