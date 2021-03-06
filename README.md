
# dnss

dnss is a daemon for using DNS over HTTPS.

It can act as a proxy (the most common use case), and as a server (in case you
want end to end control).


[![Build Status](https://travis-ci.org/albertito/dnss.svg?branch=master)](https://travis-ci.org/albertito/dnss)
[![Go Report Card](https://goreportcard.com/badge/github.com/albertito/dnss)](https://goreportcard.com/report/github.com/albertito/dnss)


## Features

* Supports the JSON-based protocol as implemented by
  [dns.google.com](https://dns.google.com)
  ([reference](https://developers.google.com/speed/public-dns/docs/dns-over-https)).
* Supports the [DNS Queries over HTTPS
  (DoH)](https://tools.ietf.org/html/draft-ietf-doh-dns-over-https) proposed
  standard (and implemented by [Cloudflare's 1.1.1.1](https://1.1.1.1/)).
* Local cache (optional).
* HTTP(s) proxy support, autodetected from the environment.
* Monitoring HTTP server, with exported variables and tracing to help
  debugging.
* Separate resolution for specific domains, useful for home networks with
  local DNS servers.


## Install

### Debian/Ubuntu

The `dnss` package installs the daemon configured in proxy mode and ready to
use, using Google's public resolvers (and easily changed via configuration).

```
$ sudo apt install dnss
```


### Manual install

```
# If you have Go installed but no environment prepared, do:
mkdir /tmp/dnss; export GOPATH=/tmp/dnss; cd $GOPATH

# Download and build the binary.
go get blitiri.com.ar/go/dnss

# Copy the binary to a system-wide location.
sudo cp $GOPATH/bin/dnss /usr/local/bin

# Set it up in systemd.
sudo cp $GOPATH/src/blitiri.com.ar/go/dnss/etc/systemd/dns-to-https/* \
	/etc/systemd/system/

sudo systemctl dnss enable
```
