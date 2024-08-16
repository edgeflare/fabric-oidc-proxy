develop: run from repository root

```shell
go run ./example-ccaas/...
```

build
```shell
docker build -t registry.example.org/yourusername/your-cc:tag . -f example-ccaas/Dockerfile
# docker build -t docker.io/edgeflare/fabric-ccaas-example:asset_cc . -f example-ccaas/Dockerfile
```