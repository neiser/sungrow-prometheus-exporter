{ lib, buildGoModule }:

buildGoModule {
  pname = "sungrow-prometheus-exporter";
  version = "0.2.0";

  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [ ../go.mod ../go.sum ../src ../config ];
  };

  vendorHash = "sha256-qmW3eQkOl19H6bR44gLar9nPTQcodefAnq0/RvhhfhQ=";

  subPackages = [ "src" ];

  postInstall = ''
    mv $out/bin/src $out/bin/sungrow-prometheus-exporter
  '';

  meta = {
    description = "Prometheus exporter for Sungrow hybrid inverters via Modbus TCP";
    homepage = "https://github.com/neiser/sungrow-prometheus-exporter";
    platforms = lib.platforms.linux;
    mainProgram = "sungrow-prometheus-exporter";
  };
}
