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

Open the web UI with a browser! It is at [http://localhost:8000](http://localhost:8000)
by default.

You can also send commands to the API directly:
```shell-session
Switch to the side-by-side scene on the projector stage
$ curl http://localhost:8000/api/scene/projector/side-by-side
```

Limited keyboard shortcuts are also available:
- Use the digit keys to switch between scenes
- Use `Ctrl-Shift-q` to exit

## Development

### Core development

Fazantix is written in Go with most of the code residing in `lib/`

### Web API

The Web API uses Swagger, and is built automatically by the makefile. The
relevant source is in `lib/api`. The automatically-generated API documentation
can be accessed by loading the `/swagger` URL in a browser.

### Web UI

The Web UI is built using [vite](https://vite.dev/) and resides in `web_ui/`.
A `build.sh` script is provided for building the UI into a single `index.html`
file that is then included in fazantix's builtin web server by default.

This, however, is not suitable for development because the bundled `index.html`
is, well, bundled, and also minified, and thus difficult to debug. In order
to start a development webserver to aid with Web UI development, use
`./web_ui/build.sh serve` and then point your browser at [http://localhost:5173](http://localhost:5173).
The development web server will proxy the api connections to a locally-running
fazantix at port 8000, but you can also override the `FAZANTIX_URL` environment
variable to make the locally-hosted web UI connect to a remote fazantix instance.

The development web server will refresh the webpage or show errors whenever
the code is modified.
