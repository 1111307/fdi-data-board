FROM artifactory.momenta.works/docker-momenta/fleet/fdi-buildbase:v1.0.3 AS builder

COPY . /src
WORKDIR /src


RUN sed -i "s|https://dl-cdn.alpinelinux.org/alpine|https://artifactory.momenta.works/artifactory/alpine-remote|g" /etc/apk/repositories \
    && sed -i "s|https://mirrors.tuna.tsinghua.edu.cn/alpine|https://artifactory.momenta.works/artifactory/alpine-remote|g" /etc/apk/repositories \
    && apk update  \
    && apk add --no-cache tzdata \
    && apk add --no-cache gawk


# 构建工具统一用 go install pkg@version 安装在项目模块之外,禁止在 /src 内裸 go get:
# 项目目录内的 go get 会解析上游依赖图元数据(retract 列表),低版本工具链遇到
# 高 go 版本要求的模块(如 anthropic-sdk-go v1.67.0)会直接报错退出。
RUN set -x\
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.3  \
	&& go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0 \
	&& go install github.com/favadi/protoc-go-inject-tag@latest \
	&& go install github.com/swaggo/swag/cmd/swag@v1.16.2 \
    && go install github.com/go-kratos/kratos/cmd/kratos/v2@v2.0.0-20251205160234-b9fab9a5a5ab \
    && go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest \
    && go install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.0 \
    && go install github.com/google/wire/cmd/wire@latest

# 依赖下载与校验(不用 tidy:tidy 会重算整张依赖图并可能改动 go.mod/go.sum)
RUN set -x\
    && go mod download

RUN mkdir -p bin \
     && make all  \
    && GOOS=linux GOARCH=amd64 go build -buildvcs=false -ldflags '-w -s -extldflags "-static"' -o ./bin/server ./cmd/...  \
    && upx /src/bin/server

FROM artifactory.momenta.works/docker-momenta/alpine:3.13

RUN  sed -i "s|https://dl-cdn.alpinelinux.org/alpine|https://artifactory.momenta.works/artifactory/alpine-remote|g" /etc/apk/repositories \
    && sed -i "s|https://mirrors.tuna.tsinghua.edu.cn/alpine|https://artifactory.momenta.works/artifactory/alpine-remote|g" /etc/apk/repositories \
    && apk update  \
    && apk add --no-cache tzdata

ENV TZ=Asia/Shanghai

COPY --from=builder /src/bin/server /app/server
COPY --from=builder /src/configs/*.yaml /app/configs/
COPY --from=builder /src/docs /app/docs/

EXPOSE 8000
EXPOSE 9000
EXPOSE 8080

ENV RUN_MODE=prd MYSQL_USERNAME=root MYSQL_PASSWORD=FDPProdP@ss MYSQL_ADDR=10.10.2.27:3306
ENV KEYCLOAK_URL=https://keycloak-prod-cla.mmtwork.com/  \
    KEYCLOAK_REALM=momenta-dev
ENV REDIS_ADDRS=10.10.1.182:6379,10.10.0.8:6379,10.10.1.24:6379 REDIS_PASSWORD=Redis-Prod-P@ss
ENV CLICKHOUSE_ON=false
ENV KAFKA_BROKERS=10.10.1.106:9092,10.10.3.113:9092,10.10.3.139:9092
ENV FIS_PROBE_ON=false
ENV UM_ENDPOINT=fdi-user-manager-backend.fdi.svc.cluster.local:9000

WORKDIR /app

CMD ./server -conf ./configs/config-${RUN_MODE}.yaml
