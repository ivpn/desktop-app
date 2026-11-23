{
  description = "Development shell for building IVPN Desktop on Apple Silicon";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
  inputs.nixpkgs-legacy.url = "github:NixOS/nixpkgs/nixos-25.05"; # old libresolv needed by the macOS 12 SDK

  outputs =
    { nixpkgs, nixpkgs-legacy, ... }:
    let
      # for now test on darwin only, can add linux too
      system = "aarch64-darwin";
      pkgs = import nixpkgs { inherit system; };
      legacyPkgs = import nixpkgs-legacy { inherit system; };
      macosSdk = legacyPkgs.apple-sdk_12;
      archTarget = {
        aarch64-darwin = "arm64";
      }.${system};
      goToolchain = pkgs.go_1_26;
      nodeToolchain = pkgs.nodejs_22;

      buildCli = pkgs.writeShellScriptBin "build-cli" ''exec ./cli/References/macOS/build.sh "$@"'';
      buildDaemon = pkgs.writeShellScriptBin "build-daemon" ''exec ./daemon/References/macOS/scripts/build-all.sh -wifi "$@"'';
      # Using nixpkgs#rcodesign
      # requires an upstream change adding an explicit ad-hoc signing mode
      # so skip for now until basic merged
      buildApp = pkgs.writeShellScriptBin "build-app" ''exec ./ui/References/macOS/build.sh "$@"'';

    in
    {
      devShells.${system}.default = pkgs.mkShell {
        env = {
          ARCH_TARGET = archTarget;
          IVPN_BUILD_SKIP_PROMPT = "1";
        };

        shellHook = ''
          # exact version as used by this repo
          # no need to install xcode or xcode-select
          export DEVELOPER_DIR="${macosSdk}"
          export MACOSX_DEPLOYMENT_TARGET="12.0"
          export SDKROOT="${macosSdk.sdkroot}"
          export NIX_LDFLAGS="''${NIX_LDFLAGS//"-L${pkgs.darwin.libresolv}/lib"/"-L${legacyPkgs.darwin.libresolv}/lib"}"
        '';

        packages = with pkgs; [
          autoconf
          automake
          buildApp
          buildCli
          buildDaemon
          cacert
          cmake
          curl
          git
          goToolchain
          gnumake
          libtool
          ninja
          nodeToolchain
          pam
          perl
          pkg-config
        ];
      };
    };
}
