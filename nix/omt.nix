# Reproducible build of the Open Media Transport stack (libvmx -> libomtnet ->
# libomt), exposing libomt via pkg-config so fazantix can build with the `omt`
# build tag.  See issue #90.
#
# Dependency chain (all MIT, all github.com/openmediatransport):
#   * libvmx     - C++ VMX codec, built with clang        -> libvmx.so
#   * libomtnet  - C# netstandard2.0 OMT protocol library  -> libomtnet.dll
#   * libomt     - C# net8.0 NativeAOT C-API wrapper        -> libomt.so + libomt.h
#
# x86_64-linux only for now (aarch64 is a follow-up; the libvmx flags and the
# linux-x64 RID below are hardcoded).
{
  lib,
  stdenv,
  fetchFromGitHub,
  clang,
  buildDotnetModule,
  dotnetCorePackages,
  patchelf,
  zlib,
  openssl,
  icu,
  avahi,
}:

let
  dotnet-sdk = dotnetCorePackages.sdk_8_0;

  srcs = {
    libvmx = fetchFromGitHub {
      owner = "openmediatransport";
      repo = "libvmx";
      rev = "f73569e767b9d9177519bf5765c9434dfe8af51f";
      hash = "sha256-wK/06ZP9D3khX9rNBaGQok/rOsd0riwoFduLd9S65jw=";
    };
    libomtnet = fetchFromGitHub {
      owner = "openmediatransport";
      repo = "libomtnet";
      rev = "b8566e9bfb37c354b651548b4e68e7e7e31e90e3";
      hash = "sha256-YC6y+RJe4pU+26GzRpeWasaE0Fwo+g+2nAKB2Jf+kLI=";
    };
    libomt = fetchFromGitHub {
      owner = "openmediatransport";
      repo = "libomt";
      rev = "cf1f48036247208847fa8513026f99182ad01f57";
      hash = "sha256-2nbkimRl6l5DO6Zl6hlztGqQ+hIXTE8x25oMvb/wvYA=";
    };
  };

  # libomt.so is a self-contained NativeAOT image (NEEDED is only libc/libm/
  # libpthread) but it dlopens these by name at runtime, so put them on its
  # (forced, dlopen-honoured) RPATH:
  #   * libvmx          - VMX codec (P/Invoke "libvmx")
  #   * openssl, icu    - .NET runtime crypto/globalization
  #   * avahi           - Linux service discovery (P/Invoke "libavahi-*")
  runtimeRpath = lib.makeLibraryPath [
    libvmx
    openssl
    icu
    zlib
    avahi
  ];

  # ilc itself is a CoreCLR app that dlopens these while running.
  ilcRunDeps = lib.makeLibraryPath [
    openssl
    icu
    zlib
    stdenv.cc.cc.lib
  ];

  libvmx = stdenv.mkDerivation {
    pname = "libvmx";
    version = "0-unstable-2025-06-22";
    src = srcs.libvmx;

    nativeBuildInputs = [ clang ];

    buildPhase = ''
      runHook preBuild
      cd build
      sh ./buildlinuxx64.sh
      runHook postBuild
    '';

    installPhase = ''
      runHook preInstall
      install -Dm755 libvmx.so $out/lib/libvmx.so
      runHook postInstall
    '';

    meta = {
      description = "VMX codec (Open Media Transport)";
      homepage = "https://github.com/openmediatransport/libvmx";
      license = lib.licenses.mit;
      platforms = [ "x86_64-linux" ];
    };
  };

  libomtnet = buildDotnetModule {
    pname = "libomtnet";
    version = "1.0.0.16";
    src = srcs.libomtnet;

    inherit dotnet-sdk;
    runtimeId = "linux-x64";

    projectFile = "libomtnet.csproj";
    nugetDeps = ./libomtnet-deps.json;

    # net40 cannot be built with the dotnet SDK on Linux; keep only the
    # netstandard2.0 target which is what libomt links against.
    postPatch = ''
      substituteInPlace libomtnet.csproj \
        --replace-fail 'netstandard2.0;net40' 'netstandard2.0'
    '';

    # It's a library, not an app: just grab the built assembly.
    executables = [ ];
    installPhase = ''
      runHook preInstall
      install -Dm644 "$(find bin -name libomtnet.dll | head -n1)" $out/lib/libomtnet.dll
      runHook postInstall
    '';

    meta = {
      description = "Open Media Transport .NET library";
      homepage = "https://github.com/openmediatransport/libomtnet";
      license = lib.licenses.mit;
      platforms = [ "x86_64-linux" ];
    };
  };

  libomt = buildDotnetModule {
    pname = "libomt";
    version = "1.0.0.16";
    src = srcs.libomt;

    inherit dotnet-sdk;
    runtimeId = "linux-x64";

    # Publish the csproj directly (not the .sln) so buildDotnetModule passes
    # --runtime linux-x64, which NativeAOT requires.
    projectFile = "libomt.csproj";
    nugetDeps = ./libomt-deps.json;

    # NativeAOT is inherently self-contained.
    selfContainedBuild = true;
    useAppHost = false;

    nativeBuildInputs = [
      clang # used by ILC to link the final native image
      patchelf
    ];
    buildInputs = [ zlib ];

    # Point the libomtnet <Reference> HintPath at our built assembly instead of
    # the upstream sibling-directory layout. The NativeAOT ILCompiler toolchain
    # is provided (already NixOS-patched) by the nixpkgs dotnet SDK, so no
    # explicit PackageReference is needed.
    postPatch = ''
      substituteInPlace libomt.csproj \
        --replace-fail '..\libomtnet\bin\Release\netstandard2.0\libomtnet.dll' \
                       '${libomtnet}/lib/libomtnet.dll'
    '';

    # nixpkgs already ships a NixOS-patched ILCompiler (the restored package
    # carries a `.nix-patched` marker), so no patchelf of `ilc` is needed here.
    # We only run the ilc process in invariant-globalization mode and make ICU/
    # OpenSSL reachable, mirroring what the ilc CoreCLR runtime expects.
    preBuild = ''
      export DOTNET_SYSTEM_GLOBALIZATION_INVARIANT=1
      export LD_LIBRARY_PATH="${ilcRunDeps}''${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
    '';

    executables = [ ];

    postInstall = ''
      mkdir -p $out/include $out/lib/pkgconfig
      mv $out/lib/libomt/libomt.so $out/lib/libomt.so
      cp libomt.h $out/include/libomt.h
      rm -rf $out/lib/libomt

      {
        echo "prefix=$out"
        echo "libdir=$out/lib"
        echo "includedir=$out/include"
        echo ""
        echo "Name: libomt"
        echo "Description: Open Media Transport C library (NativeAOT)"
        echo "Version: 1.0.0.16"
        echo "Libs: -L$out/lib -lomt"
        echo "Cflags: -I$out/include"
      } > $out/lib/pkgconfig/libomt.pc
    '';

    # Set the runtime RPATH after the generic fixup phase, which would otherwise
    # shrink away these entries (libvmx/openssl/icu are dlopened by name, not in
    # DT_NEEDED). force-rpath uses DT_RPATH, which glibc honours for dlopen from
    # within libomt.so.
    postFixup = ''
      patchelf --force-rpath --set-rpath "${runtimeRpath}" $out/lib/libomt.so
    '';

    meta = {
      description = "Open Media Transport C library (NativeAOT) with pkg-config";
      homepage = "https://github.com/openmediatransport/libomt";
      license = lib.licenses.mit;
      platforms = [ "x86_64-linux" ];
    };
  };
in
{
  inherit libvmx libomtnet libomt;
}
