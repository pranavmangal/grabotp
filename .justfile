build-dev:
    go build -o grabotp

build-release:
    go build -ldflags "-s -w -X main.version=$(git describe --tags)" -o grabotp

clean:
    rm -rf grabotp completions dist
