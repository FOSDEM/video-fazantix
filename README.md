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

or

```shell-session
$ nix build
```

## Running

Running on bare metal

```shell-session
$ xinit ./fazantix configfile.yaml
```

Running for development

```shell-session
$ make run
```

or

```shell-session
$ make run CONFIG=fosdem.yaml
```

To quit, press Ctrl+Shift+Q.

## Control

```shell-session
Switch to the side-by-side scene on the projector stage
$ curl http://localhost:8000/api/scene/projector/side-by-side
```
