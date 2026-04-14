#!/bin/sh

# if you don't have golang installed, I recommend mise
# Reference: https://mise.jdx.dev/installing-mise.html
# curl https://mise.run | sh
# mise doctor
# mise use -g go@latest

# fetch dependencies
go mod init go-filemover

# create checksums, version info
go mod tidy

# build a local binary (omit -o to install to $GOROOT/bin/go-filemover)
go -o go-filemover.go

# replace ambiguous 'user' folder name with your own $USER name 
sed -i "s|user|$USER|g" example_config.toml

# make the config directory
mkdir -pv $HOME/.config/go-filemover

# copy the example config and rename it to default
cp -v example_config.toml $HOME/.config/go-filemover/config.toml

# edit config.toml to your spec
$EDITOR $HOME/.config/go-filemover/config.toml
