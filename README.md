# TorBlock
TorBlock is a [Traefik](https://traefik.io) plugin which can block requests originating from the Tor network. The publicly available list of Tor exit nodes (`https://check.torproject.org/exit-addresses`) is fetched regularly in order to identify the requests to be blocked.

## Configuration

Requirements: `Traefik >= v2.5.5`

### Static

For each plugin, the Traefik static configuration must define the module name (as is usual for Go packages).

The following declaration (given here in YAML) defines an plugin:

```yaml
# Static configuration
pilot:
  token: xxxxx

experimental:
  plugins:
    torblock:
      moduleName: github.com/jpxd/torblock
      version: v0.1.1
```

Here is an example of a file provider dynamic configuration (given here in YAML), where the interesting part is the `http.middlewares` section:

```yaml
# Dynamic configuration

http:
  routers:
    my-router:
      rule: host(`demo.localhost`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - my-middleware

  services:
   service-foo:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000
  
  middlewares:
    my-middleware:
      torblock:
        enabled: true
```

### Local Mode

Traefik also offers a developer mode that can be used for temporary testing or offline usage of plugins not hosted on GitHub. To use a plugin in local mode, the Traefik static configuration must define the module name (as is usual for Go packages) and a path to a [Go workspace](https://golang.org/doc/gopath_code.html#Workspaces), which can be the local GOPATH or any directory.

The plugins must be placed in `./plugins-local` directory, which should be next to the Traefik binary.
The source code of the plugin should be organized as follows:

```
./plugins-local/
    └── src
        └── github.com
            └── jpxd
                └── torblock
                    ├── torblock.go
                    ├── torblock_test.go
                    ├── go.mod
                    ├── LICENSE
                    ├── Makefile
                    └── README.md
```

```yaml
# Static configuration
pilot:
  token: xxxxx

experimental:
  localPlugins:
    example:
      moduleName: github.com/jpxd/torblock
```

(In the above example, the `torblock` plugin will be loaded from the path `./plugins-local/src/github.com/jpxd/torblock`.)

```yaml
# Dynamic configuration

http:
  routers:
    my-router:
      rule: host(`demo.localhost`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - my-middleware

  services:
   service-foo:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000
  
  middlewares:
    my-middleware:
      plugin:
        torblock:
          enabled: true
```

### Examples

You can also see a working example `docker-compose.yml` in the `examples` directory, which loads the plugin in local mode.