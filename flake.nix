{
  description = "Prometheus exporter for Sungrow hybrid inverters via Modbus TCP";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      packages.${system}.default = pkgs.callPackage ./nix/package.nix { };

      overlays.default = final: prev: {
        sungrow-prometheus-exporter = final.callPackage ./nix/package.nix { };
      };

      nixosModules.default = { pkgs, ... }: {
        imports = [ ./nix/module.nix ];
        services.sungrow-prometheus-exporter.package = nixpkgs.lib.mkDefault
          self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      };

      checks.${system}.default = self.packages.${system}.default;

      formatter.${system} = pkgs.nixpkgs-fmt;
    };
}
