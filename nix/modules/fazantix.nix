{ config, pkgs, lib, ... }:
let
  name = "fazantix";
  cfg = config.services.${name};

  uid = 4842;
  gid = uid;

  fazantdir = "/var/lib/fazantix";

  config-text = pkgs.writeTextFile {
    name = "${name}-config.json";
    text = builtins.toJSON cfg.config;
  };
  config-file = pkgs.stdenvNoCC.mkDerivation {
    name = "${name}-config-file";
    meta.description = "Config file for ${name}";
    buildInputs = [ pkgs.coreutils ];
    phases = [ "installPhase" ];
    installPhase = ''
      mkdir -p $out
      cp -vfT ${config-text} $out/${name}-config.json
      ${pkgs.fazantix-wayland}/bin/fazantix-validate-config $out/${name}-config.json
    '';
  };
in {
  options = {
    services.${name} = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = ''
          Whether to enable ${name}, a video mixer
        '';
      };

      config = lib.mkOption {
        type = lib.types.raw; # fixme, validate config properly
        description = "${name} configuration";
      };

      openApiPort = lib.mkOption {
        type = lib.types.bool;
        description = "Open a firewall port for the ${name} API";
      };

      user = lib.mkOption {
        type = lib.types.str;
        default = name;
        description = "User account under which ${name} runs.";
      };

      group = lib.mkOption {
        type = lib.types.str;
        default = name;
        description = "Group account under which ${name} runs.";
      };
    };
  };

  config = lib.mkIf cfg.enable {
    services.seatd.enable = true;

    systemd.services.${name} = {
      enable = true;
      description = "${name} video mixer";
      after = [ "seatd.service" ];
      wants = [ "seatd.service" ];
      path = [ pkgs.bash ];
      serviceConfig = {
        Type = "simple";
        ExecStart = "${pkgs.cage}/bin/cage -d -- ${pkgs.fazantix-wayland}/bin/fazantix ${config-file}/${name}-config.json";
        User = "${cfg.user}";
        Group = "${cfg.group}";

        RuntimeDirectory = name;
        RuntimeDirectoryMode = "0700";
        Environment = [
          "XDG_RUNTIME_DIR=/run/${name}"
          "LIBSEAT_BACKEND=seatd"
        ];

        StateDirectory = name;  # creates fazantdir; FIXME - make this depend on fazantdir
        WorkingDirectory = "${fazantdir}";
      };
      wantedBy = [ "multi-user.target" ];
    };

    users.users = lib.optionalAttrs (cfg.user == name) {
      ${name} = {
        inherit uid;
        group = cfg.group;
        extraGroups = [ "audio" "video" "seat" ];
        description = "${name} user";
        home = "${fazantdir}";
        isSystemUser = true;
      };
    };

    users.groups = lib.optionalAttrs (cfg.group == name) {
      ${name}.gid = gid;
    };

    networking.firewall.allowedTCPPorts = lib.optionals (
      builtins.isBool cfg.openApiPort && cfg.openApiPort
    ) [
      (lib.toIntBase10 (builtins.elemAt
        (lib.splitString ":" cfg.config.api.bind) 1))
    ];
  };
}
