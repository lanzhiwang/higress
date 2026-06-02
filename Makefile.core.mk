ifeq ($(DEBUG),1)
$(info Makefile.core.mk Shell Environment Variables1: $(shell env))
$(info *************)
$(foreach v,$(.VARIABLES),$(info $(v) = $($(v))))
$(info -----------------------------------)
endif

SHELL := /bin/bash -o pipefail

export HIGRESS_BASE_VERSION ?= 2023-07-20T20-50-43

export HUB ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/higress

export ISTIO_BASE_REGISTRY ?= $(HUB)

export BASE_VERSION ?= $(HIGRESS_BASE_VERSION)

export CHARTS ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/charts

VERSION_PACKAGE := github.com/alibaba/higress/v2/pkg/cmd/lversion

GIT_COMMIT:=$(shell git rev-parse HEAD)

GO_LDFLAGS += -X $(VERSION_PACKAGE).higressVersion=$(shell cat VERSION) \
	-X $(VERSION_PACKAGE).gitCommitID=$(GIT_COMMIT)

GO ?= go

export GOPROXY ?= https://proxy.golang.org,direct

TARGET_ARCH ?= amd64

VALID_ARCHS := amd64 arm64
ifeq ($(filter $(TARGET_ARCH),$(VALID_ARCHS)),)
  $(error "TARGET_ARCH must be one of: $(VALID_ARCHS)")
endif

GOARCH_LOCAL := $(TARGET_ARCH)
GOOS_LOCAL := $(TARGET_OS)
RELEASE_LDFLAGS='$(GO_LDFLAGS) -extldflags -static -s -w'

export OUT:=$(TARGET_OUT)
export OUT_LINUX:=$(TARGET_OUT_LINUX)

BUILDX_PLATFORM ?=

# If tag not explicitly set in users' .istiorc.mk or command line, default to the git sha.
TAG ?= $(shell git rev-parse --verify HEAD)
ifeq ($(TAG),)
  $(error "TAG cannot be empty")
endif

VARIANT :=
ifeq ($(VARIANT),)
  TAG_VARIANT:=${TAG}
else
  TAG_VARIANT:=${TAG}-${VARIANT}
endif

HIGRESS_DOCKER_BUILD_TOP:=${OUT_LINUX}/docker_build

HIGRESS_BINARIES:=./cmd/higress

HGCTL_PROJECT_DIR=./hgctl
HGCTL_BINARIES:=./cmd/hgctl

ifeq ($(DEBUG),1)
$(info Makefile.core.mk Shell Environment Variables2: $(shell env))
$(info *************)
$(foreach v,$(.VARIABLES),$(info $(v) = $($(v))))
$(info -----------------------------------)
endif

$(OUT):
	@mkdir -p $@

# OUT = /work/out/linux_amd64
# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run
# make[1]: Entering directory '/work'
# mkdir -p /work/out/linux_amd64
# make[1]: Leaving directory '/work'
# build-tools:/work#

submodule:
	git submodule update --init
#	git submodule update --remote

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run submodule
# make[1]: Entering directory '/work'
# git submodule update --init
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: prebuild
prebuild: submodule
	./tools/hack/prebuild.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run prebuild
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: default
default: build

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run default
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: go.test.coverage
go.test.coverage: prebuild
	go test ./cmd/... ./pkg/... -race -coverprofile=coverage.xml -covermode=atomic

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run go.test.coverage
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# go test ./cmd/... ./pkg/... -race -coverprofile=coverage.xml -covermode=atomic
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: build
build: prebuild $(OUT)
	GOPROXY="$(GOPROXY)" GOOS=$(GOOS_LOCAL) GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh $(OUT)/ $(HIGRESS_BINARIES)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: build-linux
build-linux: prebuild $(OUT)
	GOPROXY="$(GOPROXY)" GOOS=linux GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh $(OUT_LINUX)/ $(HIGRESS_BINARIES)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-linux
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

$(AMD64_OUT_LINUX)/higress:
	GOPROXY="$(GOPROXY)" GOOS=linux GOARCH=amd64 LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh ./out/linux_amd64/ $(HIGRESS_BINARIES)

# AMD64_OUT_LINUX = /root/huzhi/higress/out/linux_amd64
# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run $(AMD64_OUT_LINUX)/higress
# bash: AMD64_OUT_LINUX: command not found
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

$(ARM64_OUT_LINUX)/higress:
	GOPROXY="$(GOPROXY)" GOOS=linux GOARCH=arm64 LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh ./out/linux_arm64/ $(HIGRESS_BINARIES)

# ARM64_OUT_LINUX = /root/huzhi/higress/out/linux_arm64
# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run $(ARM64_OUT_LINUX)/higress
# bash: ARM64_OUT_LINUX: command not found
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# make[1]: Leaving directory '/work'
# build-tools:/work#


.PHONY: build-hgctl
build-hgctl: prebuild $(OUT)
	GOPROXY=$(GOPROXY) GOOS=$(GOOS_LOCAL) GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) PROJECT_DIR="$(HGCTL_PROJECT_DIR)" tools/hack/gobuild.sh $(OUT)/ $(HGCTL_BINARIES)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-hgctl
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY=https://proxy.golang.org GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' PROJECT_DIR="./hgctl" tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/hgctl
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: build-linux-hgctl
build-linux-hgctl: prebuild $(OUT)
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) PROJECT_DIR="$(HGCTL_PROJECT_DIR)" tools/hack/gobuild.sh $(OUT_LINUX)/ $(HGCTL_BINARIES)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-linux-hgctl
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY=https://proxy.golang.org GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' PROJECT_DIR="./hgctl" tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/hgctl
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: build-hgctl-multiarch
build-hgctl-multiarch: prebuild $(OUT)
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=amd64 LDFLAGS=$(RELEASE_LDFLAGS) PROJECT_DIR="$(HGCTL_PROJECT_DIR)" tools/hack/gobuild.sh ../out/linux_amd64/ $(HGCTL_BINARIES)
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=arm64 LDFLAGS=$(RELEASE_LDFLAGS) PROJECT_DIR="$(HGCTL_PROJECT_DIR)" tools/hack/gobuild.sh ../out/linux_arm64/ $(HGCTL_BINARIES)
	GOPROXY=$(GOPROXY) GOOS=windows GOARCH=amd64 LDFLAGS=$(RELEASE_LDFLAGS) PROJECT_DIR="$(HGCTL_PROJECT_DIR)" tools/hack/gobuild.sh ../out/windows_amd64/ $(HGCTL_BINARIES)
	GOPROXY=$(GOPROXY) GOOS=windows GOARCH=arm64 LDFLAGS=$(RELEASE_LDFLAGS) PROJECT_DIR="$(HGCTL_PROJECT_DIR)" tools/hack/gobuild.sh ../out/windows_arm64/ $(HGCTL_BINARIES)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-hgctl-multiarch
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY=https://proxy.golang.org GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' PROJECT_DIR="./hgctl" tools/hack/gobuild.sh ../out/linux_amd64/ ./cmd/hgctl
# GOPROXY=https://proxy.golang.org GOOS=linux GOARCH=arm64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' PROJECT_DIR="./hgctl" tools/hack/gobuild.sh ../out/linux_arm64/ ./cmd/hgctl
# GOPROXY=https://proxy.golang.org GOOS=windows GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' PROJECT_DIR="./hgctl" tools/hack/gobuild.sh ../out/windows_amd64/ ./cmd/hgctl
# GOPROXY=https://proxy.golang.org GOOS=windows GOARCH=arm64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' PROJECT_DIR="./hgctl" tools/hack/gobuild.sh ../out/windows_arm64/ ./cmd/hgctl
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: build-hgctl-macos-arm64
build-hgctl-macos-arm64: prebuild $(OUT)
	CGO_ENABLED=1 STATIC=0 GOPROXY=$(GOPROXY) GOOS=darwin GOARCH=arm64 PROJECT_DIR="$(HGCTL_PROJECT_DIR)" tools/hack/gobuild.sh ../out/darwin_arm64/ $(HGCTL_BINARIES)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-hgctl-macos-arm64
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# CGO_ENABLED=1 STATIC=0 GOPROXY=https://proxy.golang.org GOOS=darwin GOARCH=arm64 PROJECT_DIR="./hgctl" tools/hack/gobuild.sh ../out/darwin_arm64/ ./cmd/hgctl
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: build-hgctl-macos-amd64
build-hgctl-macos-amd64: prebuild $(OUT)
	CGO_ENABLED=1 STATIC=0 GOPROXY=$(GOPROXY) GOOS=darwin GOARCH=amd64 PROJECT_DIR="$(HGCTL_PROJECT_DIR)" tools/hack/gobuild.sh ../out/darwin_amd64/ $(HGCTL_BINARIES)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-hgctl-macos-amd64
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# CGO_ENABLED=1 STATIC=0 GOPROXY=https://proxy.golang.org GOOS=darwin GOARCH=amd64 PROJECT_DIR="./hgctl" tools/hack/gobuild.sh ../out/darwin_amd64/ ./cmd/hgctl
# make[1]: Leaving directory '/work'
# build-tools:/work#

# Create targets for OUT_LINUX/binary
# There are two use cases here:
# * Building all docker images (generally in CI). In this case we want to build everything at once, so they share work
# * Building a single docker image (generally during dev). In this case we just want to build the single binary alone
BUILD_ALL ?= true
define build-linux
.PHONY: $(OUT_LINUX)/$(shell basename $(1))
ifeq ($(BUILD_ALL),true)
$(OUT_LINUX)/$(shell basename $(1)): build-linux
else
$(OUT_LINUX)/$(shell basename $(1)): $(OUT_LINUX)
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh $(OUT_LINUX)/ -tags=$(2) $(1)
endif
endef

# define build-linux
# 	.PHONY: $(OUT_LINUX)/$(shell basename $(1))
# 	ifeq ($(BUILD_ALL),true)
# 		$(OUT_LINUX)/$(shell basename $(1)): build-linux
# 	else
# 		$(OUT_LINUX)/$(shell basename $(1)): $(OUT_LINUX)
# 			GOPROXY=$(GOPROXY) GOOS=linux GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh $(OUT_LINUX)/ -tags=$(2) $(1)
# 	endif
# endef

$(foreach bin,$(HIGRESS_BINARIES),$(eval $(call build-linux,$(bin),"")))

# HIGRESS_BINARIES = ./cmd/higress
# OUT_LINUX := /work/out/linux_amd64
# /work/out/linux_amd64/higress: build-linux

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run /work/out/linux_amd64/higress
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

# Create helper targets for each binary, like "pilot-discovery"
# As an optimization, these still build everything
$(foreach bin,$(HIGRESS_BINARIES),$(shell basename $(bin))): build

# HIGRESS_BINARIES = ./cmd/higress
# higress: build

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run higress
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

ifneq ($(OUT_LINUX),$(LOCAL_OUT))
# if we are on linux already, then this rule is handled by build-linux above, which handles BUILD_ALL variable
$(foreach bin,$(HIGRESS_BINARIES),${LOCAL_OUT}/$(shell basename $(bin))): build
endif

# OUT_LINUX := /work/out/linux_amd64
# /higress: build

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run /higress
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

.PHONY: push

# for now docker is limited to Linux compiles - why ?
include docker/docker.mk

ifeq ($(DEBUG),1)
$(info Makefile.core.mk Shell Environment Variables3: $(shell env))
$(info *************)
$(foreach v,$(.VARIABLES),$(info $(v) = $($(v))))
$(info -----------------------------------)
endif

docker-build-amd64: clean-higress docker.higress-amd64 ## Build and push amdd64 docker images to registry defined by $HUB and $TAG

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run docker-build-amd64
# make[1]: Entering directory '/work'
# rm -rf /work/out/linux_amd64 /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh ./out/linux_amd64/ ./cmd/higress
# time (
# 	mkdir -p /work/out/linux_amd64/docker_build/docker.higress-amd64 &&
# 	TARGET_ARCH=amd64 ./docker/docker-copy.sh docker/Dockerfile.higress /root/huzhi/higress/out/linux_amd64/higress /work/out/linux_amd64/docker_build/docker.higress-amd64 &&
# 	cd /work/out/linux_amd64/docker_build/docker.higress-amd64  &&
# 	docker build --build-arg BASE_VERSION=2023-07-20T20-50-43 --build-arg HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress --build-arg BASE_DISTRIBUTION= --build-arg TARGETARCH=amd64 -t higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab -f Dockerfile.higress .
# );
# make[1]: Leaving directory '/work'
# build-tools:/work#

docker-build: clean-higress docker.higress ## Build and push docker images to registry defined by $HUB and $TAG

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run docker-build
# make[1]: Entering directory '/work'
# rm -rf /work/out/linux_amd64 /work/out/linux_amd64
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# time (
# 	mkdir -p /work/out/linux_amd64/docker_build/docker.higress &&
# 	TARGET_ARCH=amd64 ./docker/docker-copy.sh docker/Dockerfile.higress /work/out/linux_amd64/higress /work/out/linux_amd64/docker_build/docker.higress &&
# 	cd /work/out/linux_amd64/docker_build/docker.higress  &&
# 	docker build --build-arg BASE_VERSION=2023-07-20T20-50-43 --build-arg HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress --build-arg BASE_DISTRIBUTION= -t higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab -f Dockerfile.higress .
# );
# make[1]: Leaving directory '/work'
# build-tools:/work#

docker-buildx-push: clean-env docker.higress-buildx

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run docker-buildx-push
# make[1]: Entering directory '/work'
# rm -rf out/
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh ./out/linux_amd64/ ./cmd/higress
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=arm64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh ./out/linux_arm64/ ./cmd/higress
# time (
# 	mkdir -p /work/out/linux_amd64/docker_build/docker.higress-buildx &&
# 	TARGET_ARCH=amd64 ./docker/docker-copy.sh docker/Dockerfile.higress /root/huzhi/higress/out/linux_amd64/higress /root/huzhi/higress/out/linux_arm64/higress /work/out/linux_amd64/docker_build/docker.higress-buildx &&
# 	cd /work/out/linux_amd64/docker_build/docker.higress-buildx  &&
# 	docker buildx create --name higress --node higress0 --platform linux/amd64,linux/arm64 --use &&
# 	docker buildx build --no-cache --platform linux/amd64,linux/arm64 --build-arg BASE_VERSION=2023-07-20T20-50-43 --build-arg HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress --build-arg BASE_DISTRIBUTION= -t higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab -f Dockerfile.higress . --push
# );
# make[1]: Leaving directory '/work'
# build-tools:/work#

export PARENT_GIT_TAG:=$(shell cat VERSION)
export PARENT_GIT_REVISION:=$(TAG)

export ENVOY_PACKAGE_URL_PATTERN?=https://github.com/higress-group/proxy/releases/download/v2.2.2/envoy-symbol-ARCH.tar.gz

build-envoy: prebuild
	./tools/hack/build-envoy.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-envoy
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# ./tools/hack/build-envoy.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-pilot: prebuild
	TARGET_ARCH=amd64 ./tools/hack/build-istio-pilot.sh
	TARGET_ARCH=arm64 ./tools/hack/build-istio-pilot.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-pilot
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# TARGET_ARCH=amd64 ./tools/hack/build-istio-pilot.sh
# TARGET_ARCH=arm64 ./tools/hack/build-istio-pilot.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-pilot-local: prebuild
	TARGET_ARCH=${TARGET_ARCH} ./tools/hack/build-istio-pilot.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-pilot-local
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# TARGET_ARCH=amd64 ./tools/hack/build-istio-pilot.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

buildx-prepare:
	docker buildx inspect multi-arch >/dev/null 2>&1 || docker buildx create --name multi-arch --platform linux/amd64,linux/arm64 --use

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run buildx-prepare
# make[1]: Entering directory '/work'
# docker buildx inspect multi-arch >/dev/null 2>&1 || docker buildx create --name multi-arch --platform linux/amd64,linux/arm64 --use
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-gateway: prebuild buildx-prepare build-golang-filter
	USE_REAL_USER=1 TARGET_ARCH=amd64 DOCKER_TARGETS="docker.proxyv2" ./tools/hack/build-istio-image.sh init
	USE_REAL_USER=1 TARGET_ARCH=arm64 DOCKER_TARGETS="docker.proxyv2" ./tools/hack/build-istio-image.sh init
	DOCKER_TARGETS="docker.proxyv2" IMG_URL="${IMG_URL}" ./tools/hack/build-istio-image.sh docker.buildx

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-gateway
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# docker buildx inspect multi-arch >/dev/null 2>&1 || docker buildx create --name multi-arch --platform linux/amd64,linux/arm64 --use
# TARGET_ARCH=amd64 ./tools/hack/build-golang-filters.sh
# TARGET_ARCH=arm64 ./tools/hack/build-golang-filters.sh
# USE_REAL_USER=1 TARGET_ARCH=amd64 DOCKER_TARGETS="docker.proxyv2" ./tools/hack/build-istio-image.sh init
# USE_REAL_USER=1 TARGET_ARCH=arm64 DOCKER_TARGETS="docker.proxyv2" ./tools/hack/build-istio-image.sh init
# DOCKER_TARGETS="docker.proxyv2" IMG_URL="higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab" ./tools/hack/build-istio-image.sh docker.buildx
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-gateway-local: prebuild $(if $(filter amd64,$(TARGET_ARCH)),build-golang-filter-amd64,build-golang-filter-arm64)
	TARGET_ARCH=${TARGET_ARCH} DOCKER_TARGETS="docker.proxyv2" ./tools/hack/build-istio-image.sh docker

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-gateway-local
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# TARGET_ARCH=amd64 ./tools/hack/build-golang-filters.sh
# TARGET_ARCH=amd64 DOCKER_TARGETS="docker.proxyv2" ./tools/hack/build-istio-image.sh docker
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-golang-filter-amd64:
	TARGET_ARCH=amd64 ./tools/hack/build-golang-filters.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-golang-filter-amd64
# make[1]: Entering directory '/work'
# TARGET_ARCH=amd64 ./tools/hack/build-golang-filters.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-golang-filter-arm64:
	TARGET_ARCH=arm64 ./tools/hack/build-golang-filters.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-golang-filter-arm64
# make[1]: Entering directory '/work'
# TARGET_ARCH=arm64 ./tools/hack/build-golang-filters.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-golang-filter:
	TARGET_ARCH=amd64 ./tools/hack/build-golang-filters.sh
	TARGET_ARCH=arm64 ./tools/hack/build-golang-filters.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-golang-filter
# make[1]: Entering directory '/work'
# TARGET_ARCH=amd64 ./tools/hack/build-golang-filters.sh
# TARGET_ARCH=arm64 ./tools/hack/build-golang-filters.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-istio: prebuild buildx-prepare
	DOCKER_TARGETS="docker.pilot" IMG_URL="${IMG_URL}" ./tools/hack/build-istio-image.sh docker.buildx

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-istio
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# docker buildx inspect multi-arch >/dev/null 2>&1 || docker buildx create --name multi-arch --platform linux/amd64,linux/arm64 --use
# DOCKER_TARGETS="docker.pilot" IMG_URL="higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab" ./tools/hack/build-istio-image.sh docker.buildx
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-istio-local: prebuild
	TARGET_ARCH=${TARGET_ARCH} DOCKER_TARGETS="docker.pilot" ./tools/hack/build-istio-image.sh docker

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-istio-local
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# TARGET_ARCH=amd64 DOCKER_TARGETS="docker.pilot" ./tools/hack/build-istio-image.sh docker
# make[1]: Leaving directory '/work'
# build-tools:/work#

build-wasmplugins:
	./tools/hack/build-wasm-plugins.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run build-wasmplugins
# make[1]: Entering directory '/work'
# ./tools/hack/build-wasm-plugins.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

pre-install:
	cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run pre-install
# make[1]: Entering directory '/work'
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# make[1]: Leaving directory '/work'
# build-tools:/work#

define create_ns
   kubectl get namespace | grep $(1) || kubectl create namespace $(1)
endef

# create_ns =    kubectl get namespace | grep $(1) || kubectl create namespace $(1)

install: pre-install
	cd helm/higress; helm dependency build
	helm install higress helm/higress -n higress-system --create-namespace --set 'global.local=true'

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run install
# make[1]: Entering directory '/work'
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# cd helm/higress; helm dependency build
# helm install higress helm/higress -n higress-system --create-namespace --set 'global.local=true'
# make[1]: Leaving directory '/work'
# build-tools:/work#

HIGRESS_LATEST_IMAGE_TAG ?= latest
ENVOY_LATEST_IMAGE_TAG ?= 4219c3d8e99adb269e7db947011eca24717882af
ISTIO_LATEST_IMAGE_TAG ?= ce494feddfa817346404090dfc598badb02efa83

install-dev: pre-install
	helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=$(TAG)' --set 'gateway.replicas=1' --set 'pilot.tag=$(ISTIO_LATEST_IMAGE_TAG)' --set 'gateway.tag=$(ENVOY_LATEST_IMAGE_TAG)' --set 'global.local=true'

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run install-dev
# make[1]: Entering directory '/work'
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=5cd6468650fe748ae675b92c0ec9871296a885ab' --set 'gateway.replicas=1' --set 'pilot.tag=ce494feddfa817346404090dfc598badb02efa83' --set 'gateway.tag=4219c3d8e99adb269e7db947011eca24717882af' --set 'global.local=true'
# make[1]: Leaving directory '/work'
# build-tools:/work#

install-dev-wasmplugin: build-wasmplugins pre-install
	helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=$(TAG)' --set 'gateway.replicas=1' --set 'pilot.tag=$(ISTIO_LATEST_IMAGE_TAG)' --set 'gateway.tag=$(ENVOY_LATEST_IMAGE_TAG)' --set 'global.local=true'  --set 'global.volumeWasmPlugins=true' --set 'global.onlyPushRouteCluster=false'

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run install-dev-wasmplugin
# make[1]: Entering directory '/work'
# ./tools/hack/build-wasm-plugins.sh
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=5cd6468650fe748ae675b92c0ec9871296a885ab' --set 'gateway.replicas=1' --set 'pilot.tag=ce494feddfa817346404090dfc598badb02efa83' --set 'gateway.tag=4219c3d8e99adb269e7db947011eca24717882af' --set 'global.local=true'  --set 'global.volumeWasmPlugins=true' --set 'global.onlyPushRouteCluster=false'
# make[1]: Leaving directory '/work'
# build-tools:/work#

uninstall:
	helm uninstall higress -n higress-system

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run uninstall
# make[1]: Entering directory '/work'
# helm uninstall higress -n higress-system
# make[1]: Leaving directory '/work'
# build-tools:/work#

upgrade: pre-install
	cd helm/higress; helm dependency build
	helm upgrade higress helm/higress -n higress-system --set 'global.local=true'

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run upgrade
# make[1]: Entering directory '/work'
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# cd helm/higress; helm dependency build
# helm upgrade higress helm/higress -n higress-system --set 'global.local=true'
# make[1]: Leaving directory '/work'
# build-tools:/work#

helm-push:
	cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
	cd helm; tar -zcf higress.tgz higress; helm push higress.tgz "oci://$(CHARTS)"

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run helm-push
# make[1]: Entering directory '/work'
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# cd helm; tar -zcf higress.tgz higress; helm push higress.tgz "oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/charts"
# make[1]: Leaving directory '/work'
# build-tools:/work#

cue = cue-gen -paths=./external/api/common-protos

gen-api: prebuild
	cd api;./gen.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run gen-api
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# cd api;./gen.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

gen-client: gen-api
	cd client; make generate-k8s-client

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run gen-client
# make[1]: Entering directory '/work'
# git submodule update --init
# ./tools/hack/prebuild.sh
# cd api;./gen.sh
# cd client; make generate-k8s-client
# make[1]: Leaving directory '/work'
# build-tools:/work#

DIRS_TO_CLEAN := $(OUT)
DIRS_TO_CLEAN += $(OUT_LINUX)

clean-higress: ## Cleans all the intermediate files and folders previously generated.
	rm -rf $(DIRS_TO_CLEAN)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run clean-higress
# make[1]: Entering directory '/work'
# rm -rf /work/out/linux_amd64 /work/out/linux_amd64
# make[1]: Leaving directory '/work'
# build-tools:/work#

clean-istio:
	rm -rf external/api
	rm -rf external/client-go
	rm -rf external/istio
	rm -rf external/pkg

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run clean-istio
# make[1]: Entering directory '/work'
# rm -rf external/api
# rm -rf external/client-go
# rm -rf external/istio
# rm -rf external/pkg
# make[1]: Leaving directory '/work'
# build-tools:/work#

clean-gateway: clean-istio
	rm -rf external/envoy
	rm -rf external/proxy
	rm -rf external/go-control-plane
	rm -rf external/package/envoy.tar.gz
	rm -rf external/package/*.so

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run clean-gateway
# make[1]: Entering directory '/work'
# rm -rf external/api
# rm -rf external/client-go
# rm -rf external/istio
# rm -rf external/pkg
# rm -rf external/envoy
# rm -rf external/proxy
# rm -rf external/go-control-plane
# rm -rf external/package/envoy.tar.gz
# rm -rf external/package/*.so
# make[1]: Leaving directory '/work'
# build-tools:/work#

clean-env:
	rm -rf out/

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run clean-env
# make[1]: Entering directory '/work'
# rm -rf out/
# make[1]: Leaving directory '/work'
# build-tools:/work#

clean-tool:
	rm -rf tools/bin

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run clean-tool
# make[1]: Entering directory '/work'
# rm -rf tools/bin
# make[1]: Leaving directory '/work'
# build-tools:/work#

clean: clean-higress clean-gateway clean-istio clean-env clean-tool

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run clean
# make[1]: Entering directory '/work'
# rm -rf /work/out/linux_amd64 /work/out/linux_amd64
# rm -rf external/api
# rm -rf external/client-go
# rm -rf external/istio
# rm -rf external/pkg
# rm -rf external/envoy
# rm -rf external/proxy
# rm -rf external/go-control-plane
# rm -rf external/package/envoy.tar.gz
# rm -rf external/package/*.so
# rm -rf out/
# rm -rf tools/bin
# make[1]: Leaving directory '/work'
# build-tools:/work#

include tools/tools.mk

ifeq ($(DEBUG),1)
$(info Makefile.core.mk Shell Environment Variables4: $(shell env))
$(info *************)
$(foreach v,$(.VARIABLES),$(info $(v) = $($(v))))
$(info -----------------------------------)
endif

include tools/lint.mk

# gateway-conformance-test runs gateway api conformance tests.
.PHONY: gateway-conformance-test
gateway-conformance-test:

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run gateway-conformance-test
# make[1]: Entering directory '/work'
# make[1]: Nothing to be done for 'gateway-conformance-test'.
# make[1]: Leaving directory '/work'
# build-tools:/work#

# higress-conformance-test-prepare prepares the environment for higress conformance tests.
.PHONY: higress-conformance-test-prepare
higress-conformance-test-prepare: $(tools/kind) delete-cluster create-cluster docker-build kube-load-image install-dev

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run higress-conformance-test-prepare
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# tools/hack/create-cluster.sh
# rm -rf /work/out/linux_amd64 /work/out/linux_amd64
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# time (mkdir -p /work/out/linux_amd64/docker_build/docker.higress && TARGET_ARCH=amd64 ./docker/docker-copy.sh docker/Dockerfile.higress /work/out/linux_amd64/higress /work/out/linux_amd64/docker_build/docker.higress && cd /work/out/linux_amd64/docker_build/docker.higress  && docker build --build-arg BASE_VERSION=2023-07-20T20-50-43 --build-arg HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress --build-arg BASE_DISTRIBUTION= -t higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab -f Dockerfile.higress . );
# if [ "5cd6468650fe748ae675b92c0ec9871296a885ab" = "latest" ]; then \
# 	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab; \
# fi
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/pilot ce494feddfa817346404090dfc598badb02efa83
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway 4219c3d8e99adb269e7db947011eca24717882af
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/docker-pull-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/docker-pull-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/docker-pull-image.sh docker.io/bitinit/eureka latest
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/docker-pull-image.sh openpolicyagent/opa 0.61.0
# tools/hack/docker-pull-image.sh curlimages/curl latest
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/kind-load-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/kind-load-image.sh docker.io/bitinit/eureka latest
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/kind-load-image.sh openpolicyagent/opa 0.61.0
# tools/hack/kind-load-image.sh curlimages/curl latest
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=5cd6468650fe748ae675b92c0ec9871296a885ab' --set 'gateway.replicas=1' --set 'pilot.tag=ce494feddfa817346404090dfc598badb02efa83' --set 'gateway.tag=4219c3d8e99adb269e7db947011eca24717882af' --set 'global.local=true'
# make[1]: Leaving directory '/work'
# build-tools:/work#

# higress-conformance-test runs ingress api conformance tests.
.PHONY: higress-conformance-test
higress-conformance-test: $(tools/kind) delete-cluster create-cluster docker-build kube-load-image install-dev run-higress-e2e-test delete-cluster

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run higress-conformance-test
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# tools/hack/create-cluster.sh
# rm -rf /work/out/linux_amd64 /work/out/linux_amd64
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# time (mkdir -p /work/out/linux_amd64/docker_build/docker.higress && TARGET_ARCH=amd64 ./docker/docker-copy.sh docker/Dockerfile.higress /work/out/linux_amd64/higress /work/out/linux_amd64/docker_build/docker.higress && cd /work/out/linux_amd64/docker_build/docker.higress  && docker build --build-arg BASE_VERSION=2023-07-20T20-50-43 --build-arg HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress --build-arg BASE_DISTRIBUTION= -t higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab -f Dockerfile.higress . );
# if [ "5cd6468650fe748ae675b92c0ec9871296a885ab" = "latest" ]; then \
# 	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab; \
# fi
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/pilot ce494feddfa817346404090dfc598badb02efa83
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway 4219c3d8e99adb269e7db947011eca24717882af
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/docker-pull-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/docker-pull-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/docker-pull-image.sh docker.io/bitinit/eureka latest
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/docker-pull-image.sh openpolicyagent/opa 0.61.0
# tools/hack/docker-pull-image.sh curlimages/curl latest
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/kind-load-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/kind-load-image.sh docker.io/bitinit/eureka latest
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/kind-load-image.sh openpolicyagent/opa 0.61.0
# tools/hack/kind-load-image.sh curlimages/curl latest
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=5cd6468650fe748ae675b92c0ec9871296a885ab' --set 'gateway.replicas=1' --set 'pilot.tag=ce494feddfa817346404090dfc598badb02efa83' --set 'gateway.tag=4219c3d8e99adb269e7db947011eca24717882af' --set 'global.local=true'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true --test-area=all --execute-tests=
# make[1]: Leaving directory '/work'
# build-tools:/work#

# higress-conformance-test-clean cleans the environment for higress conformance tests.
.PHONY: higress-conformance-test-clean
higress-conformance-test-clean: $(tools/kind) delete-cluster

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run higress-conformance-test-clean
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

# higress-wasmplugin-test-prepare prepares the environment for higress wasmplugin tests.
.PHONY: higress-wasmplugin-test-prepare
higress-wasmplugin-test-prepare: $(tools/kind) delete-cluster create-cluster docker-build kube-load-image install-dev-wasmplugin

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run higress-wasmplugin-test-prepare
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# tools/hack/create-cluster.sh
# rm -rf /work/out/linux_amd64 /work/out/linux_amd64
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# time (mkdir -p /work/out/linux_amd64/docker_build/docker.higress && TARGET_ARCH=amd64 ./docker/docker-copy.sh docker/Dockerfile.higress /work/out/linux_amd64/higress /work/out/linux_amd64/docker_build/docker.higress && cd /work/out/linux_amd64/docker_build/docker.higress  && docker build --build-arg BASE_VERSION=2023-07-20T20-50-43 --build-arg HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress --build-arg BASE_DISTRIBUTION= -t higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab -f Dockerfile.higress . );
# if [ "5cd6468650fe748ae675b92c0ec9871296a885ab" = "latest" ]; then \
# 	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab; \
# fi
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/pilot ce494feddfa817346404090dfc598badb02efa83
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway 4219c3d8e99adb269e7db947011eca24717882af
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/docker-pull-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/docker-pull-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/docker-pull-image.sh docker.io/bitinit/eureka latest
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/docker-pull-image.sh openpolicyagent/opa 0.61.0
# tools/hack/docker-pull-image.sh curlimages/curl latest
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/kind-load-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/kind-load-image.sh docker.io/bitinit/eureka latest
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/kind-load-image.sh openpolicyagent/opa 0.61.0
# tools/hack/kind-load-image.sh curlimages/curl latest
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# ./tools/hack/build-wasm-plugins.sh
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=5cd6468650fe748ae675b92c0ec9871296a885ab' --set 'gateway.replicas=1' --set 'pilot.tag=ce494feddfa817346404090dfc598badb02efa83' --set 'gateway.tag=4219c3d8e99adb269e7db947011eca24717882af' --set 'global.local=true'  --set 'global.volumeWasmPlugins=true' --set 'global.onlyPushRouteCluster=false'
# make[1]: Leaving directory '/work'
# build-tools:/work#

# higress-wasmplugin-test-prepare-skip-docker-build prepares the environment for higress wasmplugin tests without build higress docker image.
.PHONY: higress-wasmplugin-test-prepare-skip-docker-build
higress-wasmplugin-test-prepare-skip-docker-build: $(tools/kind) delete-cluster create-cluster prebuild
	@export TAG="$(HIGRESS_LATEST_IMAGE_TAG)" && \
	$(MAKE) kube-load-image && \
	$(MAKE) install-dev-wasmplugin

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run higress-wasmplugin-test-prepare-skip-docker-build
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# tools/hack/create-cluster.sh
# git submodule update --init
# ./tools/hack/prebuild.sh
# export TAG="latest" && \
# make kube-load-image && \
# make install-dev-wasmplugin
# make[2]: Entering directory '/work'
# ++ uname -m
# + LOCAL_ARCH=x86_64
# + export LOCAL_ARCH
# + [[ -n amd64 ]]
# + export TARGET_ARCH
# ++ uname
# + LOCAL_OS=Linux
# + export LOCAL_OS
# + [[ -n linux ]]
# + export TARGET_OS
# + [[ release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 == '' ]]
# + [[ build-tools == '' ]]
# + export UID
# + DOCKER_GID=1002
# + export DOCKER_GID
# ++ readlink '' /etc/localtime
# ++ sed -e 's/^.*zoneinfo\///'
# + TIMEZONE=
# + export TIMEZONE
# + export TARGET_OUT=/work/out/linux_amd64
# + TARGET_OUT=/work/out/linux_amd64
# + export TARGET_OUT_LINUX=/work/out/linux_amd64
# + TARGET_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export AMD64_OUT_LINUX=/work/out/linux_amd64
# + AMD64_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export ARM64_OUT_LINUX=/work/out/linux_arm64
# + ARM64_OUT_LINUX=/work/out/linux_arm64
# + export CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + export CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + export IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + export CONTAINER_CLI=docker
# + CONTAINER_CLI=docker
# + export 'ENV_BLOCKLIST=^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# + ENV_BLOCKLIST='^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# ++ declare -F -x
# ++ cut -d ' ' -f 3
# + export 'CONDITIONAL_HOST_MOUNTS=--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + CONDITIONAL_HOST_MOUNTS='--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + container_kubeconfig=
# + [[ -d /home/.docker ]]
# + [[ -d /home/.config/gcloud ]]
# + [[ -f /home/.gitconfig ]]
# + [[ -f /home/.netrc ]]
# + KUBECONFIG=/home/.kube/config
# + parse_KUBECONFIG /home/.kube/config
# + TMPDIR=
# + [[ /home/.kube/config =~ ([^:]*):(.*) ]]
# + add_KUBECONFIG_if_exists /home/.kube/config
# + [[ -f /home/.kube/config ]]
# + [[ 0 -eq 1 ]]
# + export BUILD_WITH_CONTAINER=0
# + BUILD_WITH_CONTAINER=0
# + [[ envfile == \e\n\v\f\i\l\e ]]
# + echo AMD64_OUT_LINUX=/work/out/linux_amd64
# + echo ARM64_OUT_LINUX=/work/out/linux_arm64
# + echo TARGET_OUT_LINUX=/work/out/linux_amd64
# + echo TARGET_OUT=/work/out/linux_amd64
# + echo TIMEZONE=
# + echo LOCAL_OS=Linux
# + echo TARGET_OS=linux
# + echo LOCAL_ARCH=x86_64
# + echo TARGET_ARCH=amd64
# + echo BUILD_WITH_CONTAINER=0
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# if [ "latest" = "latest" ]; then \
# 	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress latest; \
# fi
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress latest
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/pilot ce494feddfa817346404090dfc598badb02efa83
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway 4219c3d8e99adb269e7db947011eca24717882af
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/docker-pull-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/docker-pull-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/docker-pull-image.sh docker.io/bitinit/eureka latest
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/docker-pull-image.sh openpolicyagent/opa 0.61.0
# tools/hack/docker-pull-image.sh curlimages/curl latest
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/kind-load-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/kind-load-image.sh docker.io/bitinit/eureka latest
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/kind-load-image.sh openpolicyagent/opa 0.61.0
# tools/hack/kind-load-image.sh curlimages/curl latest
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# make[2]: Leaving directory '/work'
# make[2]: Entering directory '/work'
# ++ uname -m
# + LOCAL_ARCH=x86_64
# + export LOCAL_ARCH
# + [[ -n amd64 ]]
# + export TARGET_ARCH
# ++ uname
# + LOCAL_OS=Linux
# + export LOCAL_OS
# + [[ -n linux ]]
# + export TARGET_OS
# + [[ release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 == '' ]]
# + [[ build-tools == '' ]]
# + export UID
# + DOCKER_GID=1002
# + export DOCKER_GID
# ++ sed -e 's/^.*zoneinfo\///'
# ++ readlink '' /etc/localtime
# + TIMEZONE=
# + export TIMEZONE
# + export TARGET_OUT=/work/out/linux_amd64
# + TARGET_OUT=/work/out/linux_amd64
# + export TARGET_OUT_LINUX=/work/out/linux_amd64
# + TARGET_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export AMD64_OUT_LINUX=/work/out/linux_amd64
# + AMD64_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export ARM64_OUT_LINUX=/work/out/linux_arm64
# + ARM64_OUT_LINUX=/work/out/linux_arm64
# + export CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + export CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + export IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + export CONTAINER_CLI=docker
# + CONTAINER_CLI=docker
# + export 'ENV_BLOCKLIST=^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# + ENV_BLOCKLIST='^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# ++ declare -F -x
# ++ cut -d ' ' -f 3
# + export 'CONDITIONAL_HOST_MOUNTS=--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + CONDITIONAL_HOST_MOUNTS='--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + container_kubeconfig=
# + [[ -d /home/.docker ]]
# + [[ -d /home/.config/gcloud ]]
# + [[ -f /home/.gitconfig ]]
# + [[ -f /home/.netrc ]]
# + KUBECONFIG=/home/.kube/config
# + parse_KUBECONFIG /home/.kube/config
# + TMPDIR=
# + [[ /home/.kube/config =~ ([^:]*):(.*) ]]
# + add_KUBECONFIG_if_exists /home/.kube/config
# + [[ -f /home/.kube/config ]]
# + [[ 0 -eq 1 ]]
# + export BUILD_WITH_CONTAINER=0
# + BUILD_WITH_CONTAINER=0
# + [[ envfile == \e\n\v\f\i\l\e ]]
# + echo AMD64_OUT_LINUX=/work/out/linux_amd64
# + echo ARM64_OUT_LINUX=/work/out/linux_arm64
# + echo TARGET_OUT_LINUX=/work/out/linux_amd64
# + echo TARGET_OUT=/work/out/linux_amd64
# + echo TIMEZONE=
# + echo LOCAL_OS=Linux
# + echo TARGET_OS=linux
# + echo LOCAL_ARCH=x86_64
# + echo TARGET_ARCH=amd64
# + echo BUILD_WITH_CONTAINER=0
# ./tools/hack/build-wasm-plugins.sh
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=latest' --set 'gateway.replicas=1' --set 'pilot.tag=ce494feddfa817346404090dfc598badb02efa83' --set 'gateway.tag=4219c3d8e99adb269e7db947011eca24717882af' --set 'global.local=true'  --set 'global.volumeWasmPlugins=true' --set 'global.onlyPushRouteCluster=false'
# make[2]: Leaving directory '/work'
# make[1]: Leaving directory '/work'
# build-tools:/work#

# higress-wasmplugin-test runs ingress wasmplugin tests.
.PHONY: higress-wasmplugin-test
higress-wasmplugin-test: $(tools/kind) delete-cluster create-cluster docker-build kube-load-image install-dev-wasmplugin run-higress-e2e-test-wasmplugin delete-cluster

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run higress-wasmplugin-test
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# tools/hack/create-cluster.sh
# rm -rf /work/out/linux_amd64 /work/out/linux_amd64
# git submodule update --init
# ./tools/hack/prebuild.sh
# mkdir -p /work/out/linux_amd64
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh /work/out/linux_amd64/ ./cmd/higress
# time (mkdir -p /work/out/linux_amd64/docker_build/docker.higress && TARGET_ARCH=amd64 ./docker/docker-copy.sh docker/Dockerfile.higress /work/out/linux_amd64/higress /work/out/linux_amd64/docker_build/docker.higress && cd /work/out/linux_amd64/docker_build/docker.higress  && docker build --build-arg BASE_VERSION=2023-07-20T20-50-43 --build-arg HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress --build-arg BASE_DISTRIBUTION= -t higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab -f Dockerfile.higress . );
# if [ "5cd6468650fe748ae675b92c0ec9871296a885ab" = "latest" ]; then \
# 	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab; \
# fi
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/pilot ce494feddfa817346404090dfc598badb02efa83
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway 4219c3d8e99adb269e7db947011eca24717882af
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/docker-pull-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/docker-pull-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/docker-pull-image.sh docker.io/bitinit/eureka latest
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/docker-pull-image.sh openpolicyagent/opa 0.61.0
# tools/hack/docker-pull-image.sh curlimages/curl latest
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/kind-load-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/kind-load-image.sh docker.io/bitinit/eureka latest
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/kind-load-image.sh openpolicyagent/opa 0.61.0
# tools/hack/kind-load-image.sh curlimages/curl latest
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# ./tools/hack/build-wasm-plugins.sh
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=5cd6468650fe748ae675b92c0ec9871296a885ab' --set 'gateway.replicas=1' --set 'pilot.tag=ce494feddfa817346404090dfc598badb02efa83' --set 'gateway.tag=4219c3d8e99adb269e7db947011eca24717882af' --set 'global.local=true'  --set 'global.volumeWasmPlugins=true' --set 'global.onlyPushRouteCluster=false'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType= -wasmPluginName= --ingress-class=higress --debug=true --test-area=all --execute-tests=
# make[1]: Leaving directory '/work'
# build-tools:/work#

# higress-wasmplugin-test-skip-docker-build runs ingress wasmplugin tests without build higress docker image
.PHONY: higress-wasmplugin-test-skip-docker-build
higress-wasmplugin-test-skip-docker-build: $(tools/kind) delete-cluster create-cluster prebuild
	@export TAG="$(HIGRESS_LATEST_IMAGE_TAG)" && \
	$(MAKE) kube-load-image && \
	$(MAKE) install-dev-wasmplugin && \
	$(MAKE) run-higress-e2e-test-wasmplugin && \
	$(MAKE) delete-cluster

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run higress-wasmplugin-test-skip-docker-build
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# tools/hack/create-cluster.sh
# git submodule update --init
# ./tools/hack/prebuild.sh
# export TAG="latest" && \
# make kube-load-image && \
# make install-dev-wasmplugin && \
# make run-higress-e2e-test-wasmplugin && \
# make delete-cluster
# make[2]: Entering directory '/work'
# ++ uname -m
# + LOCAL_ARCH=x86_64
# + export LOCAL_ARCH
# + [[ -n amd64 ]]
# + export TARGET_ARCH
# ++ uname
# + LOCAL_OS=Linux
# + export LOCAL_OS
# + [[ -n linux ]]
# + export TARGET_OS
# + [[ release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 == '' ]]
# + [[ build-tools == '' ]]
# + export UID
# + DOCKER_GID=1002
# + export DOCKER_GID
# ++ readlink '' /etc/localtime
# ++ sed -e 's/^.*zoneinfo\///'
# + TIMEZONE=
# + export TIMEZONE
# + export TARGET_OUT=/work/out/linux_amd64
# + TARGET_OUT=/work/out/linux_amd64
# + export TARGET_OUT_LINUX=/work/out/linux_amd64
# + TARGET_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export AMD64_OUT_LINUX=/work/out/linux_amd64
# + AMD64_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export ARM64_OUT_LINUX=/work/out/linux_arm64
# + ARM64_OUT_LINUX=/work/out/linux_arm64
# + export CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + export CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + export IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + export CONTAINER_CLI=docker
# + CONTAINER_CLI=docker
# + export 'ENV_BLOCKLIST=^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# + ENV_BLOCKLIST='^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# ++ declare -F -x
# ++ cut -d ' ' -f 3
# + export 'CONDITIONAL_HOST_MOUNTS=--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + CONDITIONAL_HOST_MOUNTS='--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + container_kubeconfig=
# + [[ -d /home/.docker ]]
# + [[ -d /home/.config/gcloud ]]
# + [[ -f /home/.gitconfig ]]
# + [[ -f /home/.netrc ]]
# + KUBECONFIG=/home/.kube/config
# + parse_KUBECONFIG /home/.kube/config
# + TMPDIR=
# + [[ /home/.kube/config =~ ([^:]*):(.*) ]]
# + add_KUBECONFIG_if_exists /home/.kube/config
# + [[ -f /home/.kube/config ]]
# + [[ 0 -eq 1 ]]
# + export BUILD_WITH_CONTAINER=0
# + BUILD_WITH_CONTAINER=0
# + [[ envfile == \e\n\v\f\i\l\e ]]
# + echo AMD64_OUT_LINUX=/work/out/linux_amd64
# + echo ARM64_OUT_LINUX=/work/out/linux_arm64
# + echo TARGET_OUT_LINUX=/work/out/linux_amd64
# + echo TARGET_OUT=/work/out/linux_amd64
# + echo TIMEZONE=
# + echo LOCAL_OS=Linux
# + echo TARGET_OS=linux
# + echo LOCAL_ARCH=x86_64
# + echo TARGET_ARCH=amd64
# + echo BUILD_WITH_CONTAINER=0
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# if [ "latest" = "latest" ]; then \
# 	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress latest; \
# fi
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress latest
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/pilot ce494feddfa817346404090dfc598badb02efa83
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway 4219c3d8e99adb269e7db947011eca24717882af
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/docker-pull-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/docker-pull-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/docker-pull-image.sh docker.io/bitinit/eureka latest
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/docker-pull-image.sh openpolicyagent/opa 0.61.0
# tools/hack/docker-pull-image.sh curlimages/curl latest
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/kind-load-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/kind-load-image.sh docker.io/bitinit/eureka latest
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/kind-load-image.sh openpolicyagent/opa 0.61.0
# tools/hack/kind-load-image.sh curlimages/curl latest
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# make[2]: Leaving directory '/work'
# make[2]: Entering directory '/work'
# ++ uname -m
# + LOCAL_ARCH=x86_64
# + export LOCAL_ARCH
# + [[ -n amd64 ]]
# + export TARGET_ARCH
# ++ uname
# + LOCAL_OS=Linux
# + export LOCAL_OS
# + [[ -n linux ]]
# + export TARGET_OS
# + [[ release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 == '' ]]
# + [[ build-tools == '' ]]
# + export UID
# + DOCKER_GID=1002
# + export DOCKER_GID
# ++ readlink '' /etc/localtime
# ++ sed -e 's/^.*zoneinfo\///'
# + TIMEZONE=
# + export TIMEZONE
# + export TARGET_OUT=/work/out/linux_amd64
# + TARGET_OUT=/work/out/linux_amd64
# + export TARGET_OUT_LINUX=/work/out/linux_amd64
# + TARGET_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export AMD64_OUT_LINUX=/work/out/linux_amd64
# + AMD64_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export ARM64_OUT_LINUX=/work/out/linux_arm64
# + ARM64_OUT_LINUX=/work/out/linux_arm64
# + export CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + export CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + export IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + export CONTAINER_CLI=docker
# + CONTAINER_CLI=docker
# + export 'ENV_BLOCKLIST=^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# + ENV_BLOCKLIST='^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# ++ declare -F -x
# ++ cut -d ' ' -f 3
# + export 'CONDITIONAL_HOST_MOUNTS=--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + CONDITIONAL_HOST_MOUNTS='--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + container_kubeconfig=
# + [[ -d /home/.docker ]]
# + [[ -d /home/.config/gcloud ]]
# + [[ -f /home/.gitconfig ]]
# + [[ -f /home/.netrc ]]
# + KUBECONFIG=/home/.kube/config
# + parse_KUBECONFIG /home/.kube/config
# + TMPDIR=
# + [[ /home/.kube/config =~ ([^:]*):(.*) ]]
# + add_KUBECONFIG_if_exists /home/.kube/config
# + [[ -f /home/.kube/config ]]
# + [[ 0 -eq 1 ]]
# + export BUILD_WITH_CONTAINER=0
# + BUILD_WITH_CONTAINER=0
# + [[ envfile == \e\n\v\f\i\l\e ]]
# + echo AMD64_OUT_LINUX=/work/out/linux_amd64
# + echo ARM64_OUT_LINUX=/work/out/linux_arm64
# + echo TARGET_OUT_LINUX=/work/out/linux_amd64
# + echo TARGET_OUT=/work/out/linux_amd64
# + echo TIMEZONE=
# + echo LOCAL_OS=Linux
# + echo TARGET_OS=linux
# + echo LOCAL_ARCH=x86_64
# + echo TARGET_ARCH=amd64
# + echo BUILD_WITH_CONTAINER=0
# ./tools/hack/build-wasm-plugins.sh
# cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
# helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=latest' --set 'gateway.replicas=1' --set 'pilot.tag=ce494feddfa817346404090dfc598badb02efa83' --set 'gateway.tag=4219c3d8e99adb269e7db947011eca24717882af' --set 'global.local=true'  --set 'global.volumeWasmPlugins=true' --set 'global.onlyPushRouteCluster=false'
# make[2]: Leaving directory '/work'
# make[2]: Entering directory '/work'
# ++ uname -m
# + LOCAL_ARCH=x86_64
# + export LOCAL_ARCH
# + [[ -n amd64 ]]
# + export TARGET_ARCH
# ++ uname
# + LOCAL_OS=Linux
# + export LOCAL_OS
# + [[ -n linux ]]
# + export TARGET_OS
# + [[ release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 == '' ]]
# + [[ build-tools == '' ]]
# + export UID
# + DOCKER_GID=1002
# + export DOCKER_GID
# ++ readlink '' /etc/localtime
# ++ sed -e 's/^.*zoneinfo\///'
# + TIMEZONE=
# + export TIMEZONE
# + export TARGET_OUT=/work/out/linux_amd64
# + TARGET_OUT=/work/out/linux_amd64
# + export TARGET_OUT_LINUX=/work/out/linux_amd64
# + TARGET_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export AMD64_OUT_LINUX=/work/out/linux_amd64
# + AMD64_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export ARM64_OUT_LINUX=/work/out/linux_arm64
# + ARM64_OUT_LINUX=/work/out/linux_arm64
# + export CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + export CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + export IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + export CONTAINER_CLI=docker
# + CONTAINER_CLI=docker
# + export 'ENV_BLOCKLIST=^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# + ENV_BLOCKLIST='^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# ++ declare -F -x
# ++ cut -d ' ' -f 3
# + export 'CONDITIONAL_HOST_MOUNTS=--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + CONDITIONAL_HOST_MOUNTS='--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + container_kubeconfig=
# + [[ -d /home/.docker ]]
# + [[ -d /home/.config/gcloud ]]
# + [[ -f /home/.gitconfig ]]
# + [[ -f /home/.netrc ]]
# + KUBECONFIG=/home/.kube/config
# + parse_KUBECONFIG /home/.kube/config
# + TMPDIR=
# + [[ /home/.kube/config =~ ([^:]*):(.*) ]]
# + add_KUBECONFIG_if_exists /home/.kube/config
# + [[ -f /home/.kube/config ]]
# + [[ 0 -eq 1 ]]
# + export BUILD_WITH_CONTAINER=0
# + BUILD_WITH_CONTAINER=0
# + [[ envfile == \e\n\v\f\i\l\e ]]
# + echo AMD64_OUT_LINUX=/work/out/linux_amd64
# + echo ARM64_OUT_LINUX=/work/out/linux_arm64
# + echo TARGET_OUT_LINUX=/work/out/linux_amd64
# + echo TARGET_OUT=/work/out/linux_amd64
# + echo TIMEZONE=
# + echo LOCAL_OS=Linux
# + echo TARGET_OS=linux
# + echo LOCAL_ARCH=x86_64
# + echo TARGET_ARCH=amd64
# + echo BUILD_WITH_CONTAINER=0
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType= -wasmPluginName= --ingress-class=higress --debug=true --test-area=all --execute-tests=
# make[2]: Leaving directory '/work'
# make[2]: Entering directory '/work'
# ++ uname -m
# + LOCAL_ARCH=x86_64
# + export LOCAL_ARCH
# + [[ -n amd64 ]]
# + export TARGET_ARCH
# ++ uname
# + LOCAL_OS=Linux
# + export LOCAL_OS
# + [[ -n linux ]]
# + export TARGET_OS
# + [[ release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 == '' ]]
# + [[ build-tools == '' ]]
# + export UID
# + DOCKER_GID=1002
# + export DOCKER_GID
# ++ readlink '' /etc/localtime
# ++ sed -e 's/^.*zoneinfo\///'
# + TIMEZONE=
# + export TIMEZONE
# + export TARGET_OUT=/work/out/linux_amd64
# + TARGET_OUT=/work/out/linux_amd64
# + export TARGET_OUT_LINUX=/work/out/linux_amd64
# + TARGET_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export AMD64_OUT_LINUX=/work/out/linux_amd64
# + AMD64_OUT_LINUX=/work/out/linux_amd64
# ++ pwd
# + export ARM64_OUT_LINUX=/work/out/linux_arm64
# + ARM64_OUT_LINUX=/work/out/linux_arm64
# + export CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT=/work/out/linux_amd64
# + export CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
# + export IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
# + export CONTAINER_CLI=docker
# + CONTAINER_CLI=docker
# + export 'ENV_BLOCKLIST=^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# + ENV_BLOCKLIST='^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
# ++ declare -F -x
# ++ cut -d ' ' -f 3
# + export 'CONDITIONAL_HOST_MOUNTS=--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + CONDITIONAL_HOST_MOUNTS='--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
# + container_kubeconfig=
# + [[ -d /home/.docker ]]
# + [[ -d /home/.config/gcloud ]]
# + [[ -f /home/.gitconfig ]]
# + [[ -f /home/.netrc ]]
# + KUBECONFIG=/home/.kube/config
# + parse_KUBECONFIG /home/.kube/config
# + TMPDIR=
# + [[ /home/.kube/config =~ ([^:]*):(.*) ]]
# + add_KUBECONFIG_if_exists /home/.kube/config
# + [[ -f /home/.kube/config ]]
# + [[ 0 -eq 1 ]]
# + export BUILD_WITH_CONTAINER=0
# + BUILD_WITH_CONTAINER=0
# + [[ envfile == \e\n\v\f\i\l\e ]]
# + echo AMD64_OUT_LINUX=/work/out/linux_amd64
# + echo ARM64_OUT_LINUX=/work/out/linux_arm64
# + echo TARGET_OUT_LINUX=/work/out/linux_amd64
# + echo TARGET_OUT=/work/out/linux_amd64
# + echo TIMEZONE=
# + echo LOCAL_OS=Linux
# + echo TARGET_OS=linux
# + echo LOCAL_ARCH=x86_64
# + echo TARGET_ARCH=amd64
# + echo BUILD_WITH_CONTAINER=0
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# make[2]: Leaving directory '/work'
# make[1]: Leaving directory '/work'
# build-tools:/work#

# higress-wasmplugin-test-clean cleans the environment for higress wasmplugin tests.
.PHONY: higress-wasmplugin-test-clean
higress-wasmplugin-test-clean: $(tools/kind) delete-cluster

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run higress-wasmplugin-test-clean
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

# create-cluster creates a kube cluster with kind.
.PHONY: create-cluster
create-cluster: $(tools/kind)
	tools/hack/create-cluster.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run create-cluster
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/hack/create-cluster.sh
# make[1]: Leaving directory '/work'
# build-tools:/work#

# delete-cluster deletes a kube cluster.
.PHONY: delete-cluster
delete-cluster: $(tools/kind) ## Delete kind cluster.
	$(tools/kind) delete cluster --name higress

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run delete-cluster
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/kind delete cluster --name higress
# make[1]: Leaving directory '/work'
# build-tools:/work#

# kube-load-image loads a local built docker image into kube cluster.
# dubbo-provider-demo和nacos-standlone-rc3的镜像已经上传到阿里云镜像库，第一次需要先拉到本地
# docker pull registry.cn-hangzhou.aliyuncs.com/hinsteny/dubbo-provider-demo:0.0.1
# docker pull registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3:1.0.0-RC3
# If TAG is HIGRESS_LATEST_IMAGE_TAG, means we skip building higress docker image, so we need to pull the image first.
.PHONY: kube-load-image
kube-load-image: $(tools/kind) ## Install the Higress image to a kind cluster using the provided $IMAGE and $TAG.
	@if [ "$(TAG)" = "$(HIGRESS_LATEST_IMAGE_TAG)" ]; then \
		tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress $(TAG); \
	fi
	tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress $(TAG)
	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/pilot $(ISTIO_LATEST_IMAGE_TAG)
	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway $(ENVOY_LATEST_IMAGE_TAG)
	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
	tools/hack/docker-pull-image.sh docker.io/hashicorp/consul 1.16.0
	tools/hack/docker-pull-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
	tools/hack/docker-pull-image.sh docker.io/bitinit/eureka latest
	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
	tools/hack/docker-pull-image.sh openpolicyagent/opa 0.61.0
	tools/hack/docker-pull-image.sh curlimages/curl latest
	tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
	tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
	tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
	tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
	tools/hack/kind-load-image.sh docker.io/hashicorp/consul 1.16.0
	tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
	tools/hack/kind-load-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
	tools/hack/kind-load-image.sh docker.io/bitinit/eureka latest
	tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
	tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
	tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
	tools/hack/kind-load-image.sh openpolicyagent/opa 0.61.0
	tools/hack/kind-load-image.sh curlimages/curl latest
	tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
	tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run kube-load-image
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# if [ "5cd6468650fe748ae675b92c0ec9871296a885ab" = "latest" ]; then \
# 	tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab; \
# fi
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress 5cd6468650fe748ae675b92c0ec9871296a885ab
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/pilot ce494feddfa817346404090dfc598badb02efa83
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway 4219c3d8e99adb269e7db947011eca24717882af
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/docker-pull-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/docker-pull-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/docker-pull-image.sh docker.io/bitinit/eureka latest
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/docker-pull-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/docker-pull-image.sh openpolicyagent/opa 0.61.0
# tools/hack/docker-pull-image.sh curlimages/curl latest
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/dubbo-provider-demo 0.0.3-x86
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/nacos-standlone-rc3 1.0.0-RC3
# tools/hack/kind-load-image.sh docker.io/hashicorp/consul 1.16.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/httpbin 1.0.2
# tools/hack/kind-load-image.sh docker.io/charlie1380/eureka-registry-provider v0.3.0
# tools/hack/kind-load-image.sh docker.io/bitinit/eureka latest
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server 1.3.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-server v1.0
# tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/echo-body 1.0.0
# tools/hack/kind-load-image.sh openpolicyagent/opa 0.61.0
# tools/hack/kind-load-image.sh curlimages/curl latest
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
# make[1]: Leaving directory '/work'
# build-tools:/work#

# run-higress-e2e-test-setup starts to setup ingress e2e tests.
.PHONT: run-higress-e2e-test-setup
run-higress-e2e-test-setup:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true --test-area=setup

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run run-higress-e2e-test-setup
# make[1]: Entering directory '/work'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true --test-area=setup
# make[1]: Leaving directory '/work'
# build-tools:/work#

# run-higress-e2e-test starts to run ingress e2e tests.
.PHONY: run-higress-e2e-test
run-higress-e2e-test:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true --test-area=all --execute-tests=$(TEST_SHORTNAME)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run run-higress-e2e-test
# make[1]: Entering directory '/work'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true --test-area=all --execute-tests=
# make[1]: Leaving directory '/work'
# build-tools:/work#

# run-higress-e2e-test-run starts to run ingress e2e conformance tests.
.PHONY: run-higress-e2e-test-run
run-higress-e2e-test-run:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true --test-area=run --execute-tests=$(TEST_SHORTNAME)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run run-higress-e2e-test-run
# make[1]: Entering directory '/work'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true --test-area=run --execute-tests=
# make[1]: Leaving directory '/work'
# build-tools:/work#

# run-higress-e2e-test-clean starts to clean ingress e2e tests.
.PHONY: run-higress-e2e-test-clean
run-higress-e2e-test-clean:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true --test-area=clean

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run run-higress-e2e-test-clean
# make[1]: Entering directory '/work'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true --test-area=clean
# make[1]: Leaving directory '/work'
# build-tools:/work#

# run-higress-e2e-test-wasmplugin-setup starts to prepare ingress e2e tests.
.PHONY: run-higress-e2e-test-wasmplugin-setup
run-higress-e2e-test-wasmplugin-setup:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType=$(PLUGIN_TYPE) -wasmPluginName=$(PLUGIN_NAME) --ingress-class=higress --debug=true --test-area=setup

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run run-higress-e2e-test-wasmplugin-setup
# make[1]: Entering directory '/work'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType= -wasmPluginName= --ingress-class=higress --debug=true --test-area=setup
# make[1]: Leaving directory '/work'
# build-tools:/work#

# run-higress-e2e-test-wasmplugin starts to run ingress e2e tests.
.PHONY: run-higress-e2e-test-wasmplugin
run-higress-e2e-test-wasmplugin:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType=$(PLUGIN_TYPE) -wasmPluginName=$(PLUGIN_NAME) --ingress-class=higress --debug=true --test-area=all --execute-tests=$(TEST_SHORTNAME)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run run-higress-e2e-test-wasmplugin
# make[1]: Entering directory '/work'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType= -wasmPluginName= --ingress-class=higress --debug=true --test-area=all --execute-tests=
# make[1]: Leaving directory '/work'
# build-tools:/work#

# run-higress-e2e-test-wasmplugin-run starts to run ingress e2e conformance tests.
.PHONY: run-higress-e2e-test-wasmplugin-run
run-higress-e2e-test-wasmplugin-run:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType=$(PLUGIN_TYPE) -wasmPluginName=$(PLUGIN_NAME) --ingress-class=higress --debug=true --test-area=run --execute-tests=$(TEST_SHORTNAME)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run run-higress-e2e-test-wasmplugin-run
# make[1]: Entering directory '/work'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType= -wasmPluginName= --ingress-class=higress --debug=true --test-area=run --execute-tests=
# make[1]: Leaving directory '/work'
# build-tools:/work#

# run-higress-e2e-test-wasmplugin-clean starts to clean ingress e2e tests.
.PHONY: run-higress-e2e-test-wasmplugin-clean
run-higress-e2e-test-wasmplugin-clean:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType=$(PLUGIN_TYPE) -wasmPluginName=$(PLUGIN_NAME) --ingress-class=higress --debug=true --test-area=clean

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run run-higress-e2e-test-wasmplugin-clean
# make[1]: Entering directory '/work'
# echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
# echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
# echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
# kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
# go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType= -wasmPluginName= --ingress-class=higress --debug=true --test-area=clean
# make[1]: Leaving directory '/work'
# build-tools:/work#

ifeq ($(DEBUG),1)
$(info Makefile.core.mk Shell Environment Variables5: $(shell env))
$(info *************)
$(foreach v,$(.VARIABLES),$(info $(v) = $($(v))))
$(info -----------------------------------)
endif
