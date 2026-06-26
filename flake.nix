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
        omt = pkgs.callPackage ./nix/omt.nix { };
        # OMT (Open Media Transport) is currently x86_64-linux only
        omtSupported = system == "x86_64-linux";

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
          buildInputs = [
            pkgs.coreutils
            pkgs.rsync
          ];
          phases = [
            "unpackPhase"
            "installPhase"
          ];
          installPhase = ''
            mkdir -p $out
            rsync -rva ./ $out/
          '';
        };

        waylandDeps = [
          pkgs.wayland
        ];

        vulkanDeps = [
          pkgs.vulkan-headers
          pkgs.vulkan-loader
        ];

        xorgDeps = [
          pkgs.libX11.dev
          pkgs.libxcursor
          pkgs.libxrandr
          pkgs.libxinerama
          pkgs.libxi
          pkgs.libxxf86vm
        ];

        commonDeps = [
          pkgs.libxkbcommon
          pkgs.libGL
        ];

        buildDeps = [
          pkgs.golangci-lint
          pkgs.pkg-config
        ];

        mkFazantix =
          { name, tags }:
          pkgs.buildGoModule {
            name = "fazantix";
            src = ./.;

            # This currently needs to be manually updated when go.sum is changed
            vendorHash = "sha256-s2ExngP2VASjEjPp91lvnb+nxPQQUAGcqyerJYF6+2I=";
            goSum = ./go.sum;
            subPackages = [
              "cmd/fazantix"
              "cmd/fazantix-window"
              "cmd/fazantix-validate-config"
            ];

            inherit tags;

            doCheck = false; # don't check on every build, just check during check phase

            checkPhase = ''
              runHook preCheck

              export HOME=$TMPDIR
              make lint

              runHook postCheck
            '';

            nativeBuildInputs = buildDeps;
            buildInputs =
              commonDeps
              ++ lib.optional (builtins.elem "omt" tags) omt.libomt
              ++ lib.optionals (builtins.elem "wayland" tags) waylandDeps
              ++ lib.optionals (builtins.elem "vulkan" tags) vulkanDeps
              # FIXME: we should exclude xorgdeps when the wayland tag is not present,
              # but for some reason they are still required?
              ++ lib.optionals true xorgDeps;

            patchPhase = ''
              # generate docs
              ${pkgs.go-swag}/bin/swag init -g lib/api/api.go

              # generate web ui
              mkdir -p lib/api/static
              cp -rvf ${fazantix-web-ui}/* lib/api/static/
            '';

            meta.mainProgram = "fazantix";
          };
      in
      rec {
        packages = rec {
          fazantix-wayland = mkFazantix {
            name = "fazantix-wayland";
            tags = [
              "wayland"
              "vulkan"
            ]
            ++ lib.optional omtSupported "omt";
          };
          fazantix-xorg = mkFazantix {
            name = "fazantix-xorg";
            tags = [
            ]
            ++ lib.optional omtSupported "omt";
          };

          default = packages.fazantix-wayland;
        }
        // lib.optionalAttrs omtSupported {
          inherit (omt) libvmx libomtnet libomt;
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

        checks = {
          validate-examples = pkgs.stdenv.mkDerivation {
            name = "fmt-check";
            src = ./examples;
            doCheck = true;
            dontBuild = true;
            nativeBuildInputs = [ pkgs.coreutils ];
            checkPhase = ''
              for f in *.yaml; do
                echo "validating $f"
                ${packages.fazantix}/bin/fazantix-validate-config "$f"
              done
            '';
            installPhase = ''
              mkdir -p $out
              cp -rf * $out/
            '';
          };

          fazantix-check = packages.fazantix.overrideAttrs (old: {
            doCheck = true;
          });
        };

        formatter = pkgs.nixfmt-tree;
      }
    );
}
