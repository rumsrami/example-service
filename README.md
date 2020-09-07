Example Service
=================

- This is a stripped down version of the chat service and the in-memory store.
- I removed the react chat client.

Running locally
---
Clone the repo and navigate to the root folder
```
git clone git@github.com:rumsrami/example-service.git
cd example-service
```

#### Using docker-compose

1. Install docker-cli and docker-compose
2. From the root folder run the client and the server (client removed)
``` make compose-up ```
3. Use postman to ping and get app version
``` http://0.0.0.0:9000/rpc/Chat/Ping ```
``` http://0.0.0.0:9000/rpc/Chat/Version ```
- > The server will run and listen to requests
- > Maps ports 9000:9000
- > `make compose-down` to teardown the created containers and network


#### Pulumi
- Pulumi/index.js -> is an example how to build AWS infrastructure for such app

#### .aws folder and .github
- Show the use of github action for CD