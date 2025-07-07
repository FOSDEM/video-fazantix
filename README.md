# Fazantix vision mixer

Fazant Fazant Fazant

## Building

### Debian

```shell-session
$ apt install golang libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libglx-dev libgl-dev libxxf86vm-dev
$ make
```

### NixOS

```shell-session
$ nix develop -c make fazantix-wayland
```

## Running

```shell-session
Running on bare metal
$ xinit ./fazantix configfile.yaml

Running for development
$ make run
or
$ make run CONFIG=fosdem.yaml
```

## Control

```shell-session
Switch to the side-by-side scene on the projector stage
$ curl http://localhost:8000/api/scene/projector/side-by-side
```
