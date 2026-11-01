{ config, lib, ... }:

let
  cfg = config.services.sungrow-prometheus-exporter;
in
{
  options.services.sungrow-prometheus-exporter = {
    enable = lib.mkEnableOption "the Sungrow Modbus TCP Prometheus exporter";

    package = lib.mkOption {
      type = lib.types.package;
      description = "Exporter package to use.";
    };

    inverterHost = lib.mkOption {
      type = lib.types.str;
      example = "sungrow.grub.internal";
      description = "Host name or IP address of the inverter's Modbus TCP interface.";
    };

    inverterPort = lib.mkOption {
      type = lib.types.port;
      default = 502;
      description = "Modbus TCP port of the inverter.";
    };

    listenAddress = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1";
      description = ''
        Address the HTTP endpoint listens on. It serves the metrics and the
        actuators, which write to the inverter.
      '';
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8080;
      description = "Port the HTTP endpoint listens on.";
    };

    extraArgs = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      description = "Extra command line arguments for the exporter.";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.sungrow-prometheus-exporter = {
      description = "Prometheus exporter for the Sungrow inverter (Modbus TCP)";
      wantedBy = [ "multi-user.target" ];
      wants = [ "network-online.target" ];
      after = [ "network-online.target" ];
      serviceConfig = {
        ExecStart = lib.escapeShellArgs ([
          (lib.getExe cfg.package)
          "--inverter-address"
          "${cfg.inverterHost}:${toString cfg.inverterPort}"
          "--listen-address"
          "${cfg.listenAddress}:${toString cfg.port}"
        ] ++ cfg.extraArgs);
        DynamicUser = true;
        Restart = "always";
        RestartSec = 10;
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = true;
        RestrictAddressFamilies = [ "AF_INET" "AF_INET6" ];
      };
    };
  };
}
