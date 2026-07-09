PLUGIN_NAME ?= hello-world
BUILDER_REGISTRY ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/
REGISTRY ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/
GO_VERSION ?= 1.24.4
TINYGO_VERSION ?= 0.29.0
ORAS_VERSION ?= 1.0.0
HIGRESS_VERSION ?= 1.0.0-rc
USE_HIGRESS_TINYGO ?= false
BUILDER ?= ${BUILDER_REGISTRY}wasm-go-builder:go${GO_VERSION}-oras${ORAS_VERSION}
BUILD_TIME := $(shell date "+%Y%m%d-%H%M%S")
COMMIT_ID := $(shell git rev-parse --short HEAD 2>/dev/null)
IMAGE_TAG = $(if $(strip $(PLUGIN_VERSION)),${PLUGIN_VERSION},${BUILD_TIME}-${COMMIT_ID})
IMG ?= ${REGISTRY}${PLUGIN_NAME}:${IMAGE_TAG}
GOPROXY := $(shell go env GOPROXY)

.DEFAULT:
build:
	DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=${PLUGIN_NAME} \
	                            --build-arg BUILDER=${BUILDER}  \
	                            --build-arg GOPROXY=$(GOPROXY) \
	                            -t ${IMG} \
	                            --output extensions/${PLUGIN_NAME} \
	                            .
	@echo ""
	@echo "output wasm file: extensions/${PLUGIN_NAME}/plugin.wasm"

# $ PLUGIN_NAME=ai-token make build -n --dry-run
# DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=ai-token \
# 	                            --build-arg BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.4-oras1.0.0  \
# 	                            --build-arg GOPROXY=https://goproxy.cn,direct \
# 	                            -t higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-token:20260707-170605-35df208a \
# 	                            --output extensions/ai-token \
# 	                            .
# echo ""
# echo "output wasm file: extensions/ai-token/plugin.wasm"
# $

build-image:
	DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=${PLUGIN_NAME} \
	                            --build-arg BUILDER=${BUILDER}  \
	                            --build-arg GOPROXY=$(GOPROXY) \
	                            -t ${IMG} \
	                            .
	@echo ""
	@echo "image:            ${IMG}"

# $ PLUGIN_NAME=ai-zdtc-token make build-image -n --dry-run
# DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=ai-zdtc-token \
# 	                            --build-arg BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.4-oras1.0.0  \
# 	                            --build-arg GOPROXY=https://goproxy.cn,direct \
# 	                            -t higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-zdtc-token:20260708-172732-35df208a \
# 	                            .
# echo ""
# echo "image:            higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-zdtc-token:20260708-172732-35df208a"
# $

build-push: build-image
	docker push ${IMG}

# $ PLUGIN_NAME=ai-token make build-push -n --dry-run
# DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=ai-token \
# 	                            --build-arg BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.4-oras1.0.0  \
# 	                            --build-arg GOPROXY=https://goproxy.cn,direct \
# 	                            -t higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-token:20260707-171114-35df208a \
# 	                            .
# echo ""
# echo "image:            higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-token:20260707-171114-35df208a"
# docker push higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-token:20260707-171114-35df208a
# $

# builder:
# To build a wasm-go-builder image.
# e.g.
#   REGISTRY=<your_docker_registry> make builder
# If you want to use Go/TinyGo/Oras with another version, please modify GO_VERSION/TINYGO_VERSION/ORAS_VERSION.
# After your wasm-go-builder image is built, you can use it to build plugin image.
# e.g.
#   PLUGIN_NAME=request-block BUILDER=<your-wasm-go-builder> make
builder:
	docker buildx build --no-cache \
			--platform linux/amd64,linux/arm64 \
			--build-arg BASE_IMAGE=docker.io/ubuntu \
			--build-arg GO_VERSION=$(GO_VERSION) \
			--build-arg ORAS_VERSION=$(ORAS_VERSION) \
			--build-arg HIGRESS_VERSION=$(HIGRESS_VERSION) \
			--build-arg USE_HIGRESS_TINYGO=$(USE_HIGRESS_TINYGO) \
			-f DockerfileBuilder \
			-t ${BUILDER} \
			--push \
			.
	@echo ""
	@echo "image: ${BUILDER}"

# $ PLUGIN_NAME=ai-token make builder -n --dry-run
# docker buildx build --no-cache \
# 			--platform linux/amd64,linux/arm64 \
# 			--build-arg BASE_IMAGE=docker.io/ubuntu \
# 			--build-arg GO_VERSION=1.24.4 \
# 			--build-arg ORAS_VERSION=1.0.0 \
# 			--build-arg HIGRESS_VERSION=1.0.0-rc \
# 			--build-arg USE_HIGRESS_TINYGO=false \
# 			-f DockerfileBuilder \
# 			-t higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.4-oras1.0.0 \
# 			--push \
# 			.
# echo ""
# echo "image: higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.4-oras1.0.0"
# $

local-build:
	cd extensions/${PLUGIN_NAME};GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ./main.wasm .

	@echo ""
	@echo "wasm: extensions/${PLUGIN_NAME}/main.wasm"

# $ PLUGIN_NAME=ai-token make local-build -n --dry-run
# cd extensions/ai-token;GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ./main.wasm .
# echo ""
# echo "wasm: extensions/ai-token/main.wasm"
# $

local-run:
	python3 .devcontainer/gen_config.py ${PLUGIN_NAME}
	envoy -c extensions/${PLUGIN_NAME}/config.yaml --concurrency 0 --log-level info --component-log-level wasm:debug

# $ PLUGIN_NAME=ai-token make local-run -n --dry-run
# python3 .devcontainer/gen_config.py ai-token
# envoy -c extensions/ai-token/config.yaml --concurrency 0 --log-level info --component-log-level wasm:debug
# $

local-all: local-build local-run

# $ PLUGIN_NAME=ai-token make local-all -n --dry-run
# cd extensions/ai-token;GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ./main.wasm .
# echo ""
# echo "wasm: extensions/ai-token/main.wasm"
# python3 .devcontainer/gen_config.py ai-token
# envoy -c extensions/ai-token/config.yaml --concurrency 0 --log-level info --component-log-level wasm:debug
# $
