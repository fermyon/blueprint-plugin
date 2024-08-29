# Spin Blueprint Plugin

This is a plugin that helps to visualize the different components within a Spin application.

# Installation

## Install the latest version of the plugin

The latest stable release of the blueprint plugin can be installed like so:

```sh
spin plugins update
spin plugin install blueprint
```

## Install the canary version of the plugin

The canary release of the command trigger plugin represents the most recent commits on `main` and may not be stable, with some features still in progress.

```sh
spin plugins install --url https://github.com/fermyon/blueprint-plugin/releases/download/canary/blueprint.json
```

## Install from a local build

Alternatively, use the `spin pluginify` plugin to install from a fresh build. This will use the pluginify manifest (`spin-pluginify.toml`) to package the plugin and proceed to install it:

```sh
spin plugins install pluginify
go build -o blueprint main.go
spin pluginify --install
```

# Usage

This plugin will read a `spin.toml` file within the same directory--or whatever path specified in the `--file` flag--and output tables detailing the Spin application as a whole, as well as individual components.

## See all available commands and flags:

```sh
spin blueprint --help
```

## Show all components

If in your terminal you are in the same directory as a spin.toml file:

```sh
spin blueprint show
```

If your spin.toml file is somewhere else:

```sh

spin blueprint show --file path/to/spin.toml
```

## Show a specific component

If in your terminal you are in the same directory as a spin.toml file:

```sh
spin blueprint show component-name
```

If your spin.toml file is somewhere else:

```sh
spin blueprint show --file path/to/spin.toml component-name
```

## Loading environment variables

You can pass environment variables directly:

```sh
SPIN_VARIABLE_FOO=bar spin blueprint show --file path/to/spin.toml
```

Or read a `.env` file:

```sh
spin blueprint --env path/to/file.env show component-name
```