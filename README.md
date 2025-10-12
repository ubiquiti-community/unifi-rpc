# UniFi RPC Server

[![Keep a Changelog](https://img.shields.io/badge/changelog-Keep%20a%20Changelog-%23E05735)](CHANGELOG.md)
[![GitHub Release](https://img.shields.io/github/v/release/ubiquiti-community/unifi-rpc)](https://github.com/ubiquiti-community/unifi-rpc/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/ubiquiti-community/unifi-rpc.svg)](https://pkg.go.dev/github.com/ubiquiti-community/unifi-rpc)
[![go.mod](https://img.shields.io/github/go-mod/go-version/ubiquiti-community/unifi-rpc)](go.mod)

A BMC (Baseboard Management Controller) RPC server for UniFi switches, providing PoE port power management via SSH, compatible with Tinkerbell's bmclib and hardware provider patterns.

## Overview

This server provides BMC-style power management capabilities for UniFi switches through direct SSH control of PoE ports. It's designed to integrate seamlessly with Tinkerbell's hardware provisioning system and follows bmclib's RPC provider patterns.

## Key Features

- **Direct SSH Control**: Communicates directly with UniFi switches via SSH using `swctrl` commands
- **Port-Based Routing**: Machine identification via HTTP headers (X-Port header)
- **SSH Key Authentication**: Secure authentication using SSH private keys
- **Standard Library**: Minimal external dependencies
- **Tinkerbell Compatible**: Works with Tinkerbell's StaticHeaders configuration
- **bmclib Integration**: Compatible with bmclib's RPC provider patterns

## Architecture

### Header-Based API

```
POST /
Headers:
  X-Port: 3
Body:
  {"method": "power.get", "id": 1}
```

The server translates RPC calls into SSH commands executed on the UniFi switch:

```bash
# Power on port 3
swctrl poe set auto id 3

# Power off port 3
swctrl poe set off id 3

# Power cycle port 3
swctrl poe restart id 3

# Get port status
swctrl poe show id 3
```

## Installation

```bash
go build -o unifi-rpc ./cmd/bmc
```

## Configuration

Configuration is handled through command line flags, environment variables, or a YAML config file using Viper.

### Environment Variables

All environment variables are prefixed with `UNIFI_RPC_`:

| Environment Variable        | Default     | Description                       |
| --------------------------- | ----------- | --------------------------------- |
| `UNIFI_RPC_PORT`            | `5000`      | Port to listen on                 |
| `UNIFI_RPC_ADDRESS`         | `0.0.0.0`   | Address to listen on              |
| `UNIFI_RPC_SWITCH_HOST`     | (required)  | UniFi switch IP or hostname       |
| `UNIFI_RPC_SSH_PORT`        | `22`        | SSH port on the UniFi switch      |
| `UNIFI_RPC_SSH_USERNAME`    | `root`      | SSH username                      |
| `UNIFI_RPC_SSH_KEY_PATH`    | (required)  | Path to SSH private key file      |

### Command Line Flags

| Flag              | Default     | Description                       |
| ----------------- | ----------- | --------------------------------- |
| `--port`          | `5000`      | Port to listen on                 |
| `--address`       | `0.0.0.0`   | Address to listen on              |
| `--switch-host`   | (required)  | UniFi switch IP or hostname       |
| `--ssh-port`      | `22`        | SSH port on the UniFi switch      |
| `--ssh-username`  | `root`      | SSH username                      |
| `--ssh-key-path`  | (required)  | Path to SSH private key file      |
| `--config`        |             | Config file path (optional)       |
| `--help`          |             | Show help message                 |

### SSH Key Setup

The server requires SSH key-based authentication to the UniFi switch:

1. **Generate an SSH key** (if you don't have one):
   ```bash
   ssh-keygen -t rsa -b 4096 -f ~/.ssh/unifi_rsa -N ""
   ```

2. **Copy the public key to the UniFi switch**:
   ```bash
   ssh-copy-id -i ~/.ssh/unifi_rsa.pub root@<switch-ip>
   ```
   
   Or manually add the public key to the switch's `/root/.ssh/authorized_keys`

3. **Test SSH access**:
   ```bash
   ssh -i ~/.ssh/unifi_rsa root@<switch-ip>
   ```

### Configuration Examples

**Using environment variables:**
```bash
export UNIFI_RPC_SWITCH_HOST="10.0.24.136"
export UNIFI_RPC_SSH_KEY_PATH="~/.ssh/unifi_rsa"
export UNIFI_RPC_PORT="8080"
./unifi-rpc
```

**Using command line flags:**
```bash
./unifi-rpc \
  --switch-host="10.0.24.136" \
  --ssh-key-path="~/.ssh/unifi_rsa" \
  --port=8080
```

**Using config file (config.yaml):**
```yaml
port: 5000
address: "0.0.0.0"
switch_host: "10.0.24.136"
ssh_port: 22
ssh_username: "root"
ssh_key_path: "~/.ssh/unifi_rsa"
```

Then run:
```bash
./unifi-rpc --config=config.yaml
```

**Mixed approach (environment + flags):**
```bash
export UNIFI_RPC_SWITCH_HOST="10.0.24.136"
export UNIFI_RPC_SSH_KEY_PATH="~/.ssh/unifi_rsa"
./unifi-rpc --port=8080
```

## Usage

### Start the Server

```bash
# With environment variables
export UNIFI_RPC_SWITCH_HOST="10.0.24.136"
export UNIFI_RPC_SSH_KEY_PATH="~/.ssh/unifi_rsa"
./unifi-rpc

# With command line flags  
./unifi-rpc --switch-host="10.0.24.136" --ssh-key-path="~/.ssh/unifi_rsa" --port=8080

# Show help
./unifi-rpc --help
```

### RPC Methods

The server supports JSON-RPC 2.0 calls to the root endpoint (`/`).

All requests require the `X-Port` header to specify which port to control.

#### Supported Methods

| Method       | Parameters                          | Description                     |
| ------------ | ----------------------------------- | ------------------------------- |
| `power.get`  | none                                | Get current power state         |
| `power.set`  | `{"state": "on\|off\|cycle"}`      | Set power state                 |
| `ping`       | none                                | Health check (returns "pong")   |

**Note:** The `soft` and `reset` states are mapped to `off` and `cycle` respectively for compatibility.

### Example Requests

```bash
# Get power status for port 3
curl -X POST http://localhost:5000/ \
  -H "X-Port: 3" \
  -H "Content-Type: application/json" \
  -d '{"method": "power.get", "id": 1}'

# Response:
# {"id":1,"result":"on"}

# Power on port 3
curl -X POST http://localhost:5000/ \
  -H "X-Port: 3" \
  -H "Content-Type: application/json" \
  -d '{"method": "power.set", "params": {"state": "on"}, "id": 2}'

# Response:
# {"id":2,"result":"ok"}

# Power cycle port 3
curl -X POST http://localhost:5000/ \
  -H "X-Port: 3" \
  -H "Content-Type: application/json" \
  -d '{"method": "power.set", "params": {"state": "cycle"}, "id": 3}'

# Health check
curl -X POST http://localhost:5000/ \
  -H "X-Port: 3" \
  -H "Content-Type: application/json" \
  -d '{"method": "ping", "id": 4}'

# Response:
# {"id":4,"result":"pong"}
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
    name: unifi-port-3
---
apiVersion: bmc.tinkerbell.org/v1alpha1  
kind: Machine
metadata:
  name: unifi-port-3
spec:
  connection:
    host: unifi-rpc-server:5000
    providerOptions:
      rpc:
        consumerURL: http://unifi-rpc-server:5000
        request:
          staticHeaders:
            X-Port: ["3"]
```

**Key differences from UniFi Controller-based approach:**

- Uses `X-Port` header instead of `X-MAC-Address`
- Direct SSH connection to switch (no UniFi Controller needed)
- Port number identifies the connected device

## bmclib Integration

Compatible with bmclib's RPC provider:

```go
import "github.com/bmc-toolbox/bmclib/v2"

client := bmclib.NewClient("unifi-rpc-server", "", "",
    bmclib.WithRPCOpt(rpc.Provider{
        ConsumerURL: "http://unifi-rpc-server:5000",
        Opts: rpc.Opts{
            Request: rpc.RequestOpts{
                StaticHeaders: http.Header{
                    "X-Port": []string{"3"},
                },
            },
        },
    }),
)

// Power on the device connected to port 3
err := client.SetPowerState("on")
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

### From UniFi Controller API to SSH

This version uses direct SSH control instead of the UniFi Controller API.

**Key Changes:**

1. **Authentication**: SSH key-based instead of API keys or username/password
2. **Headers**: Uses `X-Port` only (no `X-MAC-Address` needed)
3. **Port Mapping**: Devices are identified by switch port number, not MAC address
4. **Configuration**: Requires switch hostname and SSH credentials

**Migration Steps:**

1. **Set up SSH access** to your UniFi switch (see SSH Key Setup above)

2. **Update configuration** from Controller-based to SSH-based:
   ```bash
   # Old (Controller-based)
   export UNIFI_RPC_API_KEY="..."
   export UNIFI_RPC_API_ENDPOINT="https://controller:8443"
   
   # New (SSH-based)
   export UNIFI_RPC_SWITCH_HOST="10.0.24.136"
   export UNIFI_RPC_SSH_KEY_PATH="~/.ssh/unifi_rsa"
   ```

3. **Update client headers** to use port numbers:
   ```bash
   # Old
   -H "X-MAC-Address: aa:bb:cc:dd:ee:ff" -H "X-Port: 1"
   
   # New
   -H "X-Port: 3"
   ```

4. **Map devices to ports**: Create a mapping of which device is connected to which port on your switch

5. **Update Tinkerbell/bmclib configurations** to use the new header format

**Benefits of SSH approach:**

- ✅ No UniFi Controller dependency
- ✅ Direct, faster control
- ✅ More reliable (no API rate limits)
- ✅ Simpler architecture
- ✅ Works with any UniFi switch with SSH enabled

### From Config Files to Environment Variables

If you were using a `config.yaml` file, you can still use it, or convert to environment variables:

```yaml
# config.yaml
switch_host: "10.0.24.136"
ssh_key_path: "~/.ssh/unifi_rsa"
ssh_port: 22
ssh_username: "root"
port: 5000
address: "0.0.0.0"
```

Equivalent environment variables:

```bash
export UNIFI_RPC_SWITCH_HOST="10.0.24.136"
export UNIFI_RPC_SSH_KEY_PATH="~/.ssh/unifi_rsa"
export UNIFI_RPC_SSH_PORT=22
export UNIFI_RPC_SSH_USERNAME="root"
export UNIFI_RPC_PORT=5000
export UNIFI_RPC_ADDRESS="0.0.0.0"
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
