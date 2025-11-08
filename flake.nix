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
        lib = pkgs.lib;
      in
      rec {
        packages = rec {
          fazantix-web-ui = pkgs.buildNpmPackage {
            name = "fazantix-web-ui";
            src = ./web_ui;
            npmDepsHash = "sha256-cCgiR3LxArf5uYsqo7dN8W8qiK/zAz6lkK9vcL5/ev8=";
            npmBuildScript = "build";
            installPhase = ''
              runHook preInstall
              mkdir -p $out
              cp -r dist/* $out/
              runHook postInstall
            '';
          };

          fazantix-sample-images = pkgs.stdenvNoCC.mkDerivation rec {
            name = "fazantix-sample-images";
            meta.description = "Example image files for fazantix";
            src = ./examples/images;
            buildInputs = [ pkgs.coreutils pkgs.rsync ];
            phases = [ "unpackPhase" "installPhase" ];
            installPhase = ''
              mkdir -p $out
              rsync -rva ./ $out/
            '';
          };

          fazantix = pkgs.buildGoModule {
            name = "fazantix";
            src = ./.;

            # This currently needs to be manually updated when go.sum is changed
            vendorHash = "sha256-B2+sDOcw5F7bvML90i/x3wXe2WQqzEbZuGITqlaIbRU=";
            goSum = ./go.sum;
            subPackages = [
              "cmd/fazantix"
              "cmd/fazantix-window"
              "cmd/fazantix-validate-config"
            ];

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

            patchPhase = ''
              # generate docs
              ${pkgs.go-swag}/bin/swag init -g lib/api/api.go

              # generate web ui
              mkdir -p lib/api/static
              cp -rvf ${fazantix-web-ui}/* lib/api/static/
            '';
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
