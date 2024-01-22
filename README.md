# HomeDash

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/mvdkleijn/homedash?style=for-the-badge)
[![Codacy grade](https://img.shields.io/codacy/grade/dd407766bf6249e28daa954348d5e672?style=for-the-badge)](https://app.codacy.com/gh/mvdkleijn/homedash)
[![Go Report Card](https://goreportcard.com/badge/github.com/mvdkleijn/homedash?style=for-the-badge)](https://goreportcard.com/report/github.com/mvdkleijn/homedash) [![Liberapay patrons](https://img.shields.io/liberapay/patrons/mvdkleijn?style=for-the-badge)](https://liberapay.com/mvdkleijn/) [![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/O4O7H6C73)

HomeDash is a simplistic, centralized and dynamic dashboard application for your container based home network.
It provides a REST API endpoint that allows you to add applications to the dashboard.

Features include:

- Single, statically compiled binary
- Basic UI for dashboard based on plain HTML, CSS and a sprinkling of VueJS
- Automated, regular removal/refresh of entries older that X minutes
- Configuration through environment variables or config.yml file
- Swagger docs for REST API (see http://localhost:8080/static/docs)
- Distroless container image
- Multi-architecture container image

Keep in mind that this is intended for local usage, so there are no provisions for authentication, etc.

## Usage

1) Start HomeDash by either running the container or just starting the binary;
2) Feed your HomeDash installation using either:
   - the [sidecar application](https://github.com/mvdkleijn/homedash-sidecar) or;
   - the REST API, see http://localhost:8080/static/docs/ for details.
3) Go to http://localhost:8080/ to view the dashboard. (or whatever URL you host it on)

An example docker-compose.yml file is included in the root of this git repository. The example assumes you use Traefik and something like https://github.com/tecnativa/docker-socket-proxy so adjust where needed for your situation.

## Configuration

There are three options for configuration:

1. Do nothing (see below);
2. Use a config.yml file next to the binary;
3. Use environment variables;

### Do nothing

The defaults of HomeDash are sane, though open. (see the table)

### Use config file

- Copy `config.yml.example` to `config.yml`
- Place the `config.yml` file next to the binary
- Edit where needed.

**Note:** though you *can* set CORS settings it is probably not advisable to do so unless you know what you're doing.

### Environment variables

Simply set the environment variable to the desired value. See the table below for details.

Make sure to prefix the environment variable with "HOMEDASH_".

### Table of configuration items

All environment variables **must** be prefixed by "HOMEDASH_".

| Environment variables | Config file             | Description                                          | Default                                              |
| --------------------- | ----------------------- | ---------------------------------------------------- | ---------------------------------------------------- |
| DEBUG                 | debug:                  | Output debug statements or not                       | false                                                |
| MAXAGE                | maxage:                 | Maximum age of entries from a sidecar (minutes)      | 20                                                   |
| CHECKINTERVAL         | cleancheckinterval:     | How often the server tries to clean (minutes)        | 1                                                    |
| SERVER_PORT           | server: port:           | Port to listen to                                    | "8080"                                               |
| SERVER_ADDRESS        | server: address:        | Address to listen on                                 | "" (any address)                                     |
| ICONS_TMPDIR          | icons: tmpdir:          | Location of a tmp directory used for temporary files | "./data/tmp" or "/homedash/tmp" (when container)     |
| ICONS_CACHEDIR        | icons: cachedir:        | Location of a cache directory used for caching files | "./data/cache" or "/homedash/cache" (when container) |
| CORS_DEBUG            | cors: debug:            | Show debug statements regarding CORS                 | false                                                |
| CORS_ALLOWEDHEADERS   | cors: allowedheaders:   | HTTP headers allowed by CORS                         | "Content-Type"                                       |
| CORS_ALLOWEDMETHODS   | cors: allowedmethods:   | HTTP methods allowed by CORS                         | "GET", "POST", "HEAD"                                |
| CORS_ALLOWEDORIGINS   | cors: allowedorigins:   | Origins of requests allowed by CORS                  | "*"                                                  |
| CORS_ALLOWCREDENTIALS | cors: allowcredentials: | Allow user credentials as part of request to server  | false                                                |

## Support

Supported Go versions, see: https://endoflife.date/go
Supported architectures: amd64, arm64

Source code and issues: https://github.com/mvdkleijn/homedash

## Licensing

HomeDash is made available under the [MPL-2.0](https://choosealicense.com/licenses/mpl-2.0/)
license. The full details are available from the [LICENSE](/LICENSE) file.