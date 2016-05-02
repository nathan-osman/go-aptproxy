## go-aptproxy

[![GoDoc](https://godoc.org/github.com/nathan-osman/go-aptproxy?status.svg)](https://godoc.org/github.com/nathan-osman/go-aptproxy)
[![MIT License](http://img.shields.io/badge/license-MIT-9370d8.svg?style=flat)](http://opensource.org/licenses/MIT)

This handly little proxy server includes the following features:

- avoids duplication of packages fetched from different mirrors
- provides a built-in mDNS server to advertise on the local network
- fully compatible with the `squid-deb-proxy-client` package

### Usage

The program is run as follows:

    go-aptproxy [-directory DIR] [-host HOST] [-port PORT]

By default, go-aptproxy listens on `0.0.0.0:8000` and uses `/var/cache/go-aptproxy` for storing the cached files.
