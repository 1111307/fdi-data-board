FROM artifactory.momenta.works/docker-momenta/fleet/fdi-buildbase:v1.0.2 AS builder

COPY . /src
WORKDIR /src

RUN sed -i 's/mirrors.tuna.tsinghua.edu.cn/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk update  \
    && apk add --no-cache tzdata \
    && apk add --no-cache gawk


RUN set -x\
    && go get -u google.golang.org/protobuf \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go  \
	&& go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc  \
	&& go get -u github.com/swaggo/swag/cmd/swag  \
    && go get -u github.com/favadi/protoc-go-inject-tag \
	&& go install google.golang.org/grpc/cmd/protoc-gen-go-grpc \
	&& go install github.com/swaggo/swag/cmd/swag \
	&& go install github.com/favadi/protoc-go-inject-tag \
    && go install github.com/go-kratos/kratos/cmd/kratos/v2@latest \
    && go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest \
    && go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest \
    && go install github.com/google/wire/cmd/wire@latest \
    && go mod download  \
    && go mod tidy

RUN mkdir -p bin \
    && go mod tidy \
     && make all  \
    && GOOS=linux GOARCH=amd64 go build -buildvcs=false -ldflags '-w -s -extldflags "-static"' -o ./bin/server ./cmd/...  \
    && upx /src/bin/server

FROM artifactory.momenta.works/docker-momenta/alpine:3.13

RUN apk update  \
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