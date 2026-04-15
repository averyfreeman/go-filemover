#!/bin/sh

# fetch dependencies
go mod tidy

# build a local binary
go build -o bin/go-filemover ./cmd/go-filemover/main.go

# replace ambiguous 'user' folder name with your own $USER name 
sed -i "s|user|$USER|g" example_config.toml

# make the config directory
mkdir -pv $HOME/.config/go-filemover

# copy the example config and rename it to default
cp -v example_config.toml $HOME/.config/go-filemover/config.toml

echo "Setup complete. Please edit $HOME/.config/go-filemover/config.toml to your spec."
