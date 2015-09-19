# docker-multi-tenancy
Docker Multi Tenancy Proxy

![Example](https://github.com/morfeo8marc/docker-multi-tenancy/blob/master/images/docker-tenant.png)

## Compilation

With [Captain](github.com/harbur/captain)

```
captain build
```

Or directly with Docker

```
docker build -t harbur/docker-multi-tenancy .
```

Or with Docker Compose

```
docker-compose build
```

## Getting Started

Run the Proxy using Docker:

```
docker run -P 9000:9000 -v /var/run/docker.sock:/var/run/docker.sock harbur/docker-multi-tenancy
```

Or with Docker Compose

```
docker-compose up
```


Test it with curl:

```
DOCKER_HOST=x.x.x.x
curl DOCKER_HOST:9000/images/json
```

or with Docker client:

```
DOCKER_HOST=192.168.99.100:9000
unset DOCKER_MACHINE_NAME
unset DOCKER_TLS_VERIFY
unset DOCKER_CERT_PATH
docker images
```

Now docker uses the proxy to redirect requests.
