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

### Using docker-compose

1. Install docker-cli and docker-compose
2. From the root folder run the chat service and nats.
``` 
make compose-up 
```
- > The server will run and listen to requests
- > Maps ports 9000:9000
3. Use postman to ping and get app version
``` 
http://0.0.0.0:9000/rpc/Chat/Ping
http://0.0.0.0:9000/rpc/Chat/Version 
```
4. Teardown the created containers and network
```
make compose-down
``` 


### Pulumi folder
- Pulumi/index.js shows how to provision AWS infrastructure for the service

### aws and github folders
- Show how to setup github actions