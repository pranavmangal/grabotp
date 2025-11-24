#!/bin/bash

set -e

rm -rf completions
mkdir completions

go build -o ./grabotp

./grabotp completion bash > completions/grabotp.bash
./grabotp completion zsh > completions/grabotp.zsh
./grabotp completion fish > completions/grabotp.fish
