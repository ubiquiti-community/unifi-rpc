# UniFi RPC Server

[![Keep a Changelog](https://img.shields.io/badge/changelog-Keep%20a%20Changelog-%23E05735)](CHANGELOG.md)
[![GitHub Release](https://img.shields.io/github/v/release/ubiquiti-community/unifi-rpc)](https://github.com/ubiquiti-community/unifi-rpc/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/ubiquiti-community/unifi-rpc.svg)](https://pkg.go.dev/github.com/ubiquiti-community/unifi-rpc)
[![go.mod](https://img.shields.io/github/go-mod/go-version/ubiquiti-community/unifi-rpc)](go.mod)

A header-based BMC (Baseboard Management Controller) RPC server for UniFi network devices, compatible with Tinkerbell's bmclib and hardware provider patterns.

## Overview

This server provides BMC-style power management capabilities for UniFi switches through a header-based HTTP API. It's designed to integrate seamlessly with Tinkerbell's hardware provisioning system and follows bmclib's RPC provider patterns.

## Key Features

- **Header-Based Routing**: Machine identification via HTTP headers (no path parameters)
- **Dual Authentication**: Supports both username/password and API key authentication
- **Standard Library Only**: No external routing dependencies (removed Gorilla mux)
- **Tinkerbell Compatible**: Works with Tinkerbell's StaticHeaders configuration
- **bmclib Integration**: Compatible with bmclib's RPC provider patterns

## Architecture Changes

### Before (Path-Based)

```
POST /device/{mac}/port/{port}/poweron
```

### After (Header-Based)

```
POST /poweron
Headers:
  X-MAC-Address: aa:bb:cc:dd:ee:ff
  X-Port: 1
```

## Installation

```bash
go build -o unifi-rpc ./cmd/bmc
```

## Configuration

Configuration is handled through command line flags and environment variables using Viper. No config file is required.

### Environment Variables

All environment variables are prefixed with `UNIFI_RPC_`:

| Environment Variable     | Default            | Description                            |
| ------------------------ | ------------------ | -------------------------------------- |
| `UNIFI_RPC_PORT`         | `5000`             | Port to listen on                      |
| `UNIFI_RPC_ADDRESS`      | `0.0.0.0`          | Address to listen on                   |
| `UNIFI_RPC_API_KEY`      |                    | UniFi controller API key (recommended) |
| `UNIFI_RPC_USERNAME`     |                    | UniFi controller username              |
| `UNIFI_RPC_PASSWORD`     |                    | UniFi controller password              |
| `UNIFI_RPC_API_ENDPOINT` | `https://10.0.0.1` | UniFi controller API endpoint          |
| `UNIFI_RPC_INSECURE`     | `true`             | Allow insecure TLS connections         |

### Command Line Flags

| Flag             | Default            | Description                            |
| ---------------- | ------------------ | -------------------------------------- |
| `--port`         | `5000`             | Port to listen on                      |
| `--address`      | `0.0.0.0`          | Address to listen on                   |
| `--api-key`      |                    | UniFi controller API key (recommended) |
| `--username`     |                    | UniFi controller username              |
| `--password`     |                    | UniFi controller password              |
| `--api-endpoint` | `https://10.0.0.1` | UniFi controller API endpoint          |
| `--insecure`     | `true`             | Allow insecure TLS connections         |
| `--help`         |                    | Show help message                      |

### Authentication

Choose one authentication method:

- **API Key** (recommended): Set `UNIFI_RPC_API_KEY` or use `--api-key`
- **Username/Password**: Set both `UNIFI_RPC_USERNAME` and `UNIFI_RPC_PASSWORD` or use `--username` and `--password`

### Configuration Examples

```bash
# Using environment variables
export UNIFI_RPC_API_KEY="your-api-key-here"
export UNIFI_RPC_API_ENDPOINT="https://unifi.example.com:8443"
export UNIFI_RPC_PORT="8080"
./unifi-rpc

# Using command line flags
./unifi-rpc --api-key="your-api-key-here" --api-endpoint="https://unifi.example.com:8443" --port=8080

# Using username/password
./unifi-rpc --username="admin" --password="secret" --api-endpoint="https://10.0.0.1:8443"

# Mixed approach (environment + flags)
export UNIFI_RPC_API_KEY="your-api-key-here"
./unifi-rpc --port=8080 --insecure=false
```

## Usage

### Start the Server

```bash
# With environment variables
export UNIFI_RPC_API_KEY="your-api-key-here"
./unifi-rpc

# With command line flags  
./unifi-rpc --api-key="your-api-key-here" --port=8080

# Show help
./unifi-rpc --help
```

### API Endpoints

All endpoints require these headers:

- `X-MAC-Address`: Device MAC address (e.g., "aa:bb:cc:dd:ee:ff")
- `X-Port`: Port number (e.g., "1")

#### Power Management

- `GET /status` - Get power status
- `POST /poweron` - Turn power on
- `POST /poweroff` - Turn power off
- `POST /reboot` - Reboot/power cycle

#### PXE Boot

- `POST /pxeboot` - Trigger PXE boot

#### BMC RPC (bmclib compatible)

- `POST /rpc` - Generic RPC endpoint

### Example Requests

```bash
# Check status
curl -X GET http://localhost:5000/status \
  -H "X-MAC-Address: aa:bb:cc:dd:ee:ff" \
  -H "X-Port: 1"

# Power on
curl -X POST http://localhost:5000/poweron \
  -H "X-MAC-Address: aa:bb:cc:dd:ee:ff" \
  -H "X-Port: 1"

# BMC RPC call
curl -X POST http://localhost:5000/rpc \
  -H "X-MAC-Address: aa:bb:cc:dd:ee:ff" \
  -H "X-Port: 1" \
  -H "Content-Type: application/json" \
  -d '{"method": "power.set", "params": {"state": "on"}}'
```

## Tinkerbell Integration

This server is designed to work with Tinkerbell's BMC provider system:

```yaml
# Hardware spec example
apiVersion: tinkerbell.org/v1alpha1
kind: Hardware
spec:
  bmcRef:
    apiVersion: bmc.tinkerbell.org/v1alpha1
    kind: Machine
    name: switch-01
---
apiVersion: bmc.tinkerbell.org/v1alpha1  
kind: Machine
spec:
  connection:
    host: switch-01.example.com
    providerOptions:
      rpc:
        consumerURL: http://unifi-rpc:5000
        request:
          staticHeaders:
            X-MAC-Address: ["aa:bb:cc:dd:ee:ff"]
            X-Port: ["1"]
```

## bmclib Integration

Compatible with bmclib's RPC provider:

```go
import "github.com/bmc-toolbox/bmclib/v2"

client := bmclib.NewClient("127.0.0.1", "admin", "secret",
    bmclib.WithRPCOpt(rpc.Provider{
        ConsumerURL: "http://localhost:5000",
        Opts: rpc.Opts{
            Request: rpc.RequestOpts{
                StaticHeaders: http.Header{
                    "X-MAC-Address": []string{"aa:bb:cc:dd:ee:ff"},
                    "X-Port": []string{"1"},
                },
            },
        },
    }),
)
```

[![LICENSE](https://img.shields.io/github/license/ubiquiti-community/unifi-rpc)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/ubiquiti-community/unifi-rpc/build.yml?branch=main)](https://github.com/ubiquiti-community/unifi-rpc/actions?query=workflow%3Abuild+branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/ubiquiti-community/unifi-rpc)](https://goreportcard.com/report/github.com/ubiquiti-community/unifi-rpc)
[![Codecov](https://codecov.io/gh/ubiquiti-community/unifi-rpc/branch/main/graph/badge.svg)](https://codecov.io/gh/ubiquiti-community/unifi-rpc)

⭐ `Star` this repository if you find it valuable and worth maintaining.

👁 `Watch` this repository to get notified about new releases, issues, etc.

## Description

This is a GitHub repository template for a Go application.
You can use it:

- to create a new repository with automation and environment setup,
- as reference when improving automation for an existing repository.

It includes:

- continuous integration via [GitHub Actions](https://github.com/features/actions),
- build automation via [Make](https://www.gnu.org/software/make),
- dependency management using [Go Modules](https://github.com/golang/go/wiki/Modules),
- code formatting using [gofumpt](https://github.com/mvdan/gofumpt),
- linting with [golangci-lint](https://github.com/golangci/golangci-lint)
  and [misspell](https://github.com/client9/misspell),
- unit testing with
  [race detector](https://blog.golang.org/race-detector),
  code coverage [HTML report](https://blog.golang.org/cover)
  and [Codecov report](https://codecov.io/),
- releasing using [GoReleaser](https://github.com/goreleaser/goreleaser),
- dependencies scanning and updating thanks to [Dependabot](https://dependabot.com),
- security code analysis using [CodeQL Action](https://docs.github.com/en/github/finding-security-vulnerabilities-and-errors-in-your-code/about-code-scanning),
- [Visual Studio Code](https://code.visualstudio.com) configuration with [Go](https://code.visualstudio.com/docs/languages/go) support.

## Usage

1. Sign up on [Codecov](https://codecov.io/) and configure
   [Codecov GitHub Application](https://github.com/apps/codecov) for all repositories.
1. Click the `Use this template` button (alt. clone or download this repository).
1. Replace all occurrences of `ubiquiti-community/unifi-rpc` to `your_org/repo_name` in all files.
1. Replace all occurrences of `seed` to `repo_name` in [Dockerfile](Dockerfile).
1. Update the following files:
   - [CHANGELOG.md](CHANGELOG.md)
   - [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
   - [LICENSE](LICENSE)
   - [README.md](README.md)

## Setup

Below you can find sample instructions on how to set up the development environment.
Of course, you can use other tools like [GoLand](https://www.jetbrains.com/go/),
[Vim](https://github.com/fatih/vim-go), [Emacs](https://github.com/dominikh/go-mode.el).
However, take notice that the Visual Studio Go extension is
[officially supported](https://blog.golang.org/vscode-go) by the Go team.

1. Install [Go](https://golang.org/doc/install).
1. Install [Visual Studio Code](https://code.visualstudio.com/).
1. Install [Go extension](https://code.visualstudio.com/docs/languages/go).
1. Clone and open this repository.
1. `F1` -> `Go: Install/Update Tools` -> (select all) -> OK.

## Build

### Terminal

- `make` - execute the build pipeline.
- `make help` - print help for the [Make targets](Makefile).

### Visual Studio Code

`F1` → `Tasks: Run Build Task (Ctrl+Shift+B or ⇧⌘B)` to execute the build pipeline.

## Release

The release workflow is triggered each time a tag with `v` prefix is pushed.

_CAUTION_: Make sure to understand the consequences before you bump the major version.
More info: [Go Wiki](https://github.com/golang/go/wiki/Modules#releasing-modules-v2-or-higher),
[Go Blog](https://blog.golang.org/v2-go-modules).

## Maintenance

Notable files:

- [.github/workflows](.github/workflows) - GitHub Actions workflows,
- [.github/dependabot.yml](.github/dependabot.yml) - Dependabot configuration,
- [.vscode](.vscode) - Visual Studio Code configuration files,
- [.golangci.yml](.golangci.yml) - golangci-lint configuration,
- [.goreleaser.yml](.goreleaser.yml) - GoReleaser configuration,
- [Dockerfile](Dockerfile) - Dockerfile used by GoReleaser to create a container image,
- [Makefile](Makefile) - Make targets used for development, [CI build](.github/workflows) and [.vscode/tasks.json](.vscode/tasks.json),
- [go.mod](go.mod) - [Go module definition](https://github.com/golang/go/wiki/Modules#gomod),
- [tools.go](tools.go) - [build tools](https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module).

## FAQ

### Why Visual Studio Code editor configuration

Developers that use Visual Studio Code can take advantage of the editor configuration.
While others do not have to care about it.
Setting configs for each repo is unnecessary time consuming.
VS Code is the most popular Go editor ([survey](https://blog.golang.org/survey2019-results))
and it is officially [supported by the Go team](https://blog.golang.org/vscode-go).

You can always remove the [.vscode](.vscode) directory if it really does not help you.

### Why GitHub Actions, not any other CI server

GitHub Actions is out-of-the-box if you are already using GitHub.
[Here](https://github.com/mvdan/github-actions-golang) you can learn how to use it for Go.

However, changing to any other CI server should be very simple,
because this repository has build logic and tooling installation in [Makefile](Makefile).

### How can I build on Windows

Install [tdm-gcc](https://jmeubank.github.io/tdm-gcc/)
and copy `C:\TDM-GCC-64\bin\mingw32-make.exe`
to `C:\TDM-GCC-64\bin\make.exe`.
Alternatively, you may install [mingw-w64](http://mingw-w64.org/doku.php)
and copy `mingw32-make.exe` accordingly.

Take a look [here](https://github.com/docker-archive/toolbox/issues/673#issuecomment-355275054),
if you have problems using Docker in Git Bash.

You can also use [WSL (Windows Subsystem for Linux)](https://docs.microsoft.com/en-us/windows/wsl/install-win10)
or develop inside a [Remote Container](https://code.visualstudio.com/docs/remote/containers).
However, take into consideration that then you are not going to use "bare-metal" Windows.

Consider using [goyek](https://github.com/goyek/goyek)
for creating cross-platform build pipelines in Go.

### How can I customize the release

Take a look at GoReleaser [docs](https://goreleaser.com/customization/)
as well as [its repo](https://github.com/goreleaser/goreleaser/)
how it is dogfooding its functionality.
You can use it to add deb/rpm/snap packages, Homebrew Tap, Scoop App Manifest etc.

If you are developing a library and you like handcrafted changelog and release notes,
you are free to remove any usage of GoReleaser.

## Migration Guide

### From Path-Based to Header-Based Routing

If migrating from the path-based version:

1. **Update clients** to send headers instead of path parameters
1. **Configure StaticHeaders** in Tinkerbell hardware specs
1. **Remove path parameters** from URL endpoints
1. **Update configuration** to use environment variables or command line flags instead of YAML config files
1. **Update authentication** to use API keys when possible

### From Config Files to Environment Variables

If you were using a `config.yaml` file, convert it to environment variables:

```yaml
# Old config.yaml
username: admin
password: secret
apiEndpoint: https://unifi.example.com:8443
insecure: true
```

Becomes:

```bash
# New environment variables
export UNIFI_RPC_USERNAME="admin"
export UNIFI_RPC_PASSWORD="secret"  
export UNIFI_RPC_API_ENDPOINT="https://unifi.example.com:8443"
export UNIFI_RPC_INSECURE=true
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o unifi-rpc ./cmd/bmc
```

## Contributing

Feel free to create an issue or propose a pull request.

Follow the [Code of Conduct](CODE_OF_CONDUCT.md).
