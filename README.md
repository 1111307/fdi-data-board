# Kratos Project Template

## Install Kratos

```
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
```

## Create a service

```
# 配置环境变量
export KRATOS_LAYOUT_REPO=ssh://devops.momenta.works:22/Momenta/FDI/_git/mmt-kratos-layout

# 生效环境变量(Mac，其他的系统 根据实际情况来)
source ~/.zshrc

# Create a template project
kratos new server

# 同步pb idl
make submodule
go generate ./...

```

## Generate other auxiliary files by Makefile

```
# Download and update dependencies
make init
# Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file
make api
# Generate all files
make all
```

## Automated Initialization (wire)

```
# install wire
go get github.com/google/wire/cmd/wire

# generate wire
cd cmd/server
wire
```

## Docker

```bash
# build
docker build -t <your-docker-image-name> .

# run
docker run --rm -p 8000:8000 -p 9000:9000 -e RUN_MODE=dev <your-docker-image-name>
```

## Http Swagger

访问rpc转成的http的swagger文档

```bash
http://127.0.0.1:8000/q/swagger-ui
```

## Gin Swagger

```bash
http://127.0.0.1:8080/swagger/index.html
```

## server说明

- grpc server：表示GRPC微服务的服务

- http server：表示GRPC转成Http协议的微服务

- simple server：表示无状态服务，当然也可以有状态，但是需要考虑分布式部署场景下，状态的处理方式。

- gin server: gin web框架封装的应用层服务，可以给web提供接口

