# Fazantix vision mixer

Fazant Fazant Fazant

## Building

```shell-session
$ apt install libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libglx-dev libgl-dev libxxf86vm-dev
$ make
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
