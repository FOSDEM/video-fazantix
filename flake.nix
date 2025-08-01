{
  description = "Fazant fazant fazant videomixer";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs =
    inputs:
    with inputs;
    utils.lib.eachSystem [ "x86_64-linux" "aarch64-linux" ] (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      rec {
        packages = {
          fazantix = pkgs.buildGoModule {
            name = "fazantix";
            src = ./.;

            # This currently needs to be manually updated when go.sum is changed
            vendorHash = "sha256-xuDgsIxFfiEWk9Va/tJJrVx02AUwSFjPruIkOS4ayZw=";
            goSum = ./go.sum;
            subPackages = [ "cmd/fazantix" ];

            tags = [
              "wayland"
              "vulkan"
            ];

            doCheck = false;

            nativeBuildInputs = with pkgs; [
              pkg-config
              wayland
            ];

            buildInputs = with pkgs; [
              wayland
              libxkbcommon
              vulkan-headers
              vulkan-loader
              libGL

              # FIXME: the tags specified above should probably stop this from
              # needing X11 stuff, but they still get used
              xorg.libX11.dev
            ];
          };
          default = packages.fazantix;
        };

        devShells = {
          default = pkgs.mkShell {
            inputsFrom = [ packages.fazantix ];
            buildInputs = with pkgs; [
              go
              gotools
              gopls
              golangci-lint

              cage
            ];
          };
        };

        formatter = pkgs.nixfmt-tree;
      }
    );
}
