# sungrow-prometheus-exporter

Prometheus exporter for Sungrow hybrid inverters (tested with an SH10RT) via Modbus TCP,
for example through the WiNet-S dongle. It also offers an HTTP API ("actuators") that
writes settings to the inverter.

## Usage

```
sungrow-prometheus-exporter --inverter-address <host>:502 [--listen-address :8080] [--config-dir <dir>]
```

The exporter reads the registers on each scrape of `http://<listen-address>/` (any path
except `/actuator`) and keeps the values for 500 ms. It uses Modbus unit id 1. If a
register range cannot be read, only the metrics in that range are `NaN`.

The configuration lives in [config/](config) and is embedded in the binary.
`--config-dir` reads the three files from a directory instead.

- `registers.yaml`: register addresses (1-based, as in the Sungrow documentation), types
  and value mappings
- `metrics.yaml`: the metrics, all with the prefix `sungrow_`
- `actuators.yaml`: the actuators

## Actuators

`GET /actuator` lists the actuators. `GET /actuator/<name>` reads the current value,
`POST /actuator/<name>` with the new value as body writes it and returns the value read
back. Examples:

```sh
curl http://localhost:8080/actuator/max_state_of_charge
curl -X POST http://localhost:8080/actuator/max_state_of_charge -d 0.9
curl -X POST http://localhost:8080/actuator/clock -d "$(date '+%Y-%m-%d %H:%M:%S %Z')"
```

Anyone who reaches the listen address can change the inverter settings, so listen on
localhost only.

## Nix

```sh
nix build           # the binary, with the Go tests
nix flake check
```

The flake has the package (`packages.x86_64-linux.default`, `overlays.default`) and a
NixOS module (`nixosModules.default`):

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    sungrow-prometheus-exporter.url = "github:neiser/sungrow-prometheus-exporter";
    sungrow-prometheus-exporter.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { nixpkgs, sungrow-prometheus-exporter, ... }: {
    nixosConfigurations.host = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        sungrow-prometheus-exporter.nixosModules.default
        {
          services.sungrow-prometheus-exporter = {
            enable = true;
            inverterHost = "sungrow.example";
            port = 9806;
          };
        }
      ];
    };
  };
}
```

| Option | Default | |
|---|---|---|
| `enable` | `false` | |
| `inverterHost` | | host name or IP address of the inverter |
| `inverterPort` | `502` | Modbus TCP port |
| `listenAddress` | `127.0.0.1` | address of the metrics and actuator endpoint |
| `port` | `8080` | port of the metrics and actuator endpoint |
| `extraArgs` | `[ ]` | more command line arguments |

The service runs as a dynamic user with a hardened systemd unit and restarts after
10 seconds when it fails, for example when the inverter cannot be reached at start.

## Development

```sh
go test ./...
go run ./src --inverter-address <host>:502
```
