#!/bin/sh

# build the binary using Makefile (handles init)
make build

# replace ambiguous 'user' folder name with your own $USER name 
sed -i "s|user|$USER|g" example_config.toml

# make the config directory
mkdir -pv $HOME/.config/go-filemover

# copy the example config and rename it to default
cp -v example_config.toml $HOME/.config/go-filemover/config.toml

echo "Setup complete. Please edit $HOME/.config/go-filemover/config.toml to your spec."
