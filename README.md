# Websocket Proxy

Proxy between a client and a http2 service.

A client that can not speek http2, for example it is not in an https
environment, can use this service to connect to an http2 service.

This is needed, because a browser only creates a very low number of http1.1
connections to a server (firefox and chome create 6 connections). With this
service there many long living http connections can be multiplexed via a
websocket connection.


## Install and Start

### With Go

When Go is installed, the service can be installed with:

```
go get github.com/OpenSlides/openslides-wsproxy/cmd/wsproxy
```

This command is case sensitive!

After this command, the binary is installed inside the
[gopath](https://github.com/golang/go/wiki/GOPATH). When the gopath is
integrated correctly, the service can be started with:

```
wsproxy
```


### With Checked out repository and Go

When the repository is checked out, the service can be build and started with

```
go build ./cmd/wsproxy && ./wsproxy
```


### With Checked out repository and Docker

The repository contains a dockerfile which starts the service

```
docker build . --tag openslides-wsproxy
docker run -p 9015:9015 openslides-wsproxy
```


### With prebuild docker container

There is a docker container on docker-hub:

```
docker run -p 9015:9015 ostcar/openslides-wsproxy
```


## Example requests with websocat

To test this service the tool [websocat](https://github.com/vi/websocat) can be
used.


### OpenSlides 3+

When the service is running, a websocket connection can be created with:

```
websocat ws://localhost:9015/system/autoupdate
```

Afterwards it is possible to create connections to the autoupdate service by
sending the following command via stdin throw the websocket connection. Make
sure not to close stdin or the websockte connection will also be closed.

```
{"cmd":"connect","id":1,"url":"/system/autoupdate"}
```

The service will return the data of the http2 connection. Each line of the
autoupdate-output will be one websocket frame.

To close a connection, send

```
{"cmd":"close","id":1}
```

### OpenSlides 4

The command to connect to OpenSlides4 autoupdate service needs a body:

```
{"cmd":"connect","id":1,"url":"/system/autoupdate","body":[{"ids":[5],"collection":"user","fields":{"name":null}}]}
```


## Environment variables

The service can be configured with the following environment variables:

* `WSPROXY_PORT`: Port where the service listens on. The default is `9015`.
* `WSPROXY_HOST`: The device where the service starts. The default is am
  empty string which starts the service on every device.
* `AUTOUPDATE_HOST`: Host of the autoupdate service. The default is
  `localhost`.
* `AUTOUPDATE_PORT`: Port of the autoupdate service. The default is `8002`.
* `AUTOUPDATE_PROTOCOL`: Protocol of the datastore reader. The default is
  `http`.
