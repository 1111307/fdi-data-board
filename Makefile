GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git | grep cmd))))
	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find internal -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find idl -name *.proto")
else
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find idl api -name *.proto)
	MICRO_MOD=$(shell cat go.mod|sed -n '1p'|awk '{print $$2}')
    GO_OPTS=$(shell echo $(API_PROTO_FILES) | awk '{for(i=1;i<=NF;i++){print "--go_opt=M"$$i"=$(MICRO_MOD)/"$$i}}' \
    						| awk -F/ 'OFS="/"{$$NF="";print}'	\
    						| sed 's/.$$//g')
	GO_GRPC_OPTS=$(shell echo $(API_PROTO_FILES) | awk '{for(i=1;i<=NF;i++){print "--go-grpc_opt=M"$$i"=$(MICRO_MOD)/"$$i}}' \
							| awk -F/ 'OFS="/"{$$NF="";print}'	\
							| sed 's/.$$//g')
	GO_HTTP_OPTS=$(shell echo $(API_PROTO_FILES) | awk '{for(i=1;i<=NF;i++){print "--go-http_opt=M"$$i"=$(MICRO_MOD)/"$$i}}' \
							| awk -F/ 'OFS="/"{$$NF="";print}'	\
							| sed 's/.$$//g')
endif

.PHONY: init
# init env
init:submodule
	go get -u google.golang.org/protobuf
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go get -u github.com/swaggo/swag/cmd/swag
	go get -u github.com/favadi/protoc-go-inject-tag
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/favadi/protoc-go-inject-tag
	go install github.com/swaggo/swag/cmd/swag

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=./internal \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./internal \
	       $(INTERNAL_PROTO_FILES)

.PHONY: submodule
# create submodule
submodule:
	git init
	@if [ $(shell git submodule|awk '{print $$1}') ]; then	\
		echo "submodule already exists";	\
	else	\
	  	echo "submodule not exists";	\
	  	rm -rf idl;	\
	  	git submodule add ssh://devops.momenta.works:22/Momenta/FDI/_git/cloud-idl idl;	\
	fi

.PHONY: api
# generate api proto
api:
	protoc --proto_path=. \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:. \
	       $(GO_OPTS)		\
 	       --go-http_out=paths=source_relative:. \
 	       $(GO_HTTP_OPTS)		\
 	       --go-grpc_out=paths=source_relative:. \
 	       $(GO_GRPC_OPTS)		\
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)
	protoc-go-inject-tag -input="*/*/*.pb.go"
	find api idl -name '*.pb.go' | xargs gawk -i inplace '!/ \*/ {gsub(",omitempty\"", "\"") }; { print }'

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: generate
# generate
# 不在项目模块内裸 go get / go mod tidy:会解析上游依赖图元数据,
# 低版本工具链遇到高 go 版本要求的新模块(如 anthropic-sdk-go)会报错退出
generate:
	go generate ./...

.PHONY: swagger
# 执行 swagger
swagger:
	swag fmt -d internal/server,internal/service
	swag init -g gin.go -d internal/server,internal/service,idl/common,idl,api

.PHONY: all
# generate all
all:
	make api;
	make config;
	make swagger;
	make generate;

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
