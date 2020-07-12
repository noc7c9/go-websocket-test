# Go WebSocket Test

A tiny test project to learn [Go](https://golang.org).

It's a Web server that supports creating a WebSocket connection that allows clients to:

- send `PING` messages and receive `PONG` messages in response.
- receive `CONNECTED`/`DISCONNECTED` messsages as other clients join/leave.
- send `TEXT` messages that are broadcast to other clients.

All messages are serialised to JSON.

The server also exposes an HTTP API to send messages to clients without using WebSockets.

## Server

The server is implemented in Go so you'll need to install Go.

Once Go is installed, use:
```sh
$ cd server/
$ go run main.go
```
The server will be running on `http://localhost:3000`

## Client

The client is just a static webpage, so any webserver should work.

If you have `NodeJS` installed, use:
```sh
$ cd client/
$ yarn install (OR npm install)
$ yarn start (OR npm start)
```
The client will be running on `http://localhost:8080`.

If you have `Python` installed, use:
```sh
$ cd client/
$ python -m http.server (Python 3)
$ python -m SimpleHTTPServer (Python 2)
```
The client will be running on `http://localhost:8000` in both cases.

## HTTP API

The API is used by making simple HTTP calls to the server.

### `POST /api/broadcast`

Send a message to all clients. The message must be valid JSON.

eg:
```sh
$ curl -d '{ "type": "BROADCAST", "text": "Sent to all clients" }' http://localhost:3000/api/broadcast
200 ok
```

### `GET /api/list`

List all the connected clients.

eg:
```sh
$ curl http://localhost:3000/api/list
1 connected users
xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

### `POST /api/send/:id`

Send a message to a specific clients. The message must be valid JSON.

eg:
```sh
$ curl -d '{ "type": "PRIVATE", "text": "Sent to just one client" }' http://localhost:3000/api/send/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
200 ok
```
