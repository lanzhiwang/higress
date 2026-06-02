## Copyright 2018 Istio Authors
##
## Licensed under the Apache License, Version 2.0 (the "License");
## you may not use this file except in compliance with the License.
## You may obtain a copy of the License at
##
##     http://www.apache.org/licenses/LICENSE-2.0
##
## Unless required by applicable law or agreed to in writing, software
## distributed under the License is distributed on an "AS IS" BASIS,
## WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
## See the License for the specific language governing permissions and
## limitations under the License.

docker.higress: BUILD_ARGS=--build-arg BASE_VERSION=${HIGRESS_BASE_VERSION} --build-arg HUB=${HUB}
docker.higress: $(OUT_LINUX)/higress
docker.higress: docker/Dockerfile.higress
	$(HIGRESS_DOCKER_RULE)

# HIGRESS_BASE_VERSION = 2023-07-20T20-50-43
# HUB = higress-registry.cn-hangzhou.cr.aliyuncs.com/higress
# OUT_LINUX := /work/out/linux_amd64
# HIGRESS_DOCKER_RULE = $(
# 	foreach
# 		VARIANT,
# 		$(DOCKER_BUILD_VARIANTS),
# 		time (
# 			mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ &&
# 			TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ &&
# 			cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) &&
# 			docker build $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) -t $(IMG_URL)$(call variant-tag,$(VARIANT)) -f Dockerfile.higress .
# 		);
# )

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run docker.higress
# make[1]: Entering directory '/work'
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

docker.higress-amd64: BUILD_ARGS=--build-arg BASE_VERSION=${HIGRESS_BASE_VERSION} --build-arg HUB=${HUB}
docker.higress-amd64: $(AMD64_OUT_LINUX)/higress
docker.higress-amd64: docker/Dockerfile.higress
	$(HIGRESS_DOCKER_AMD64_RULE)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run docker.higress-amd64
# make[1]: Entering directory '/work'
# GOPROXY="https://proxy.golang.org" GOOS=linux GOARCH=amd64 LDFLAGS='-X github.com/alibaba/higress/v2/pkg/cmd/lversion.higressVersion=v2.2.2 -X github.com/alibaba/higress/v2/pkg/cmd/lversion.gitCommitID=5cd6468650fe748ae675b92c0ec9871296a885ab -extldflags -static -s -w' tools/hack/gobuild.sh ./out/linux_amd64/ ./cmd/higress
# time (
# 	mkdir -p /work/out/linux_amd64/docker_build/docker.higress-amd64 &&
# 	TARGET_ARCH=amd64 ./docker/docker-copy.sh docker/Dockerfile.higress /root/huzhi/higress/out/linux_amd64/higress /work/out/linux_amd64/docker_build/docker.higress-amd64 &&
# 	cd /work/out/linux_amd64/docker_build/docker.higress-amd64  &&
# 	docker build --build-arg BASE_VERSION=2023-07-20T20-50-43 --build-arg HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress --build-arg BASE_DISTRIBUTION= --build-arg TARGETARCH=amd64 -t higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613:5cd6468650fe748ae675b92c0ec9871296a885ab -f Dockerfile.higress .
# );
# make[1]: Leaving directory '/work'
# build-tools:/work#

docker.higress-buildx: BUILD_ARGS=--build-arg BASE_VERSION=${HIGRESS_BASE_VERSION} --build-arg HUB=${HUB}
docker.higress-buildx: $(AMD64_OUT_LINUX)/higress
docker.higress-buildx: $(ARM64_OUT_LINUX)/higress
docker.higress-buildx: docker/Dockerfile.higress
	$(HIGRESS_DOCKER_BUILDX_RULE)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run docker.higress-buildx
# make[1]: Entering directory '/work'
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

# DOCKER_BUILD_VARIANTS ?=debug distroless
# Base images have two different forms:
# * "debug", suffixed as -debug. This is a ubuntu based image with a bunch of debug tools
# * "distroless", suffixed as -distroless. This is distroless image - no shell. proxyv2 uses a custom one with iptables added
# * "default", no suffix. This is currently "debug"
DOCKER_BUILD_VARIANTS ?= default
DOCKER_ALL_VARIANTS ?= debug distroless
# If INCLUDE_UNTAGGED_DEFAULT is set, then building the "DEFAULT_DISTRIBUTION" variant will publish both <tag>-<variant> and <tag>
# This can be done with DOCKER_BUILD_VARIANTS="default debug" as well, but at the expense of building twice vs building once and tagging twice
INCLUDE_UNTAGGED_DEFAULT ?= false
DEFAULT_DISTRIBUTION=debug

IMG ?= higress
IMG_URL ?= $(HUB)/$(IMG):$(TAG)

HIGRESS_DOCKER_BUILDX_RULE ?= $(foreach VARIANT,$(DOCKER_BUILD_VARIANTS), time (mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ && cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && docker buildx create --name higress --node higress0 --platform linux/amd64,linux/arm64 --use && docker buildx build --no-cache --platform linux/amd64,linux/arm64 $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) -t $(IMG_URL)$(call variant-tag,$(VARIANT)) -f Dockerfile.higress . --push  ); )

# HIGRESS_DOCKER_BUILDX_RULE = $(
# 	foreach
# 		VARIANT,
# 		$(DOCKER_BUILD_VARIANTS),
# 		time (
# 			mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ &&
# 			TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ &&
# 			cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) &&
# 			docker buildx create --name higress --node higress0 --platform linux/amd64,linux/arm64 --use &&
# 			docker buildx build --no-cache --platform linux/amd64,linux/arm64 $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) -t $(IMG_URL)$(call variant-tag,$(VARIANT)) -f Dockerfile.higress . --push
# 		);
# )

HIGRESS_DOCKER_RULE ?= $(foreach VARIANT,$(DOCKER_BUILD_VARIANTS), time (mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ && cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && docker build $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) -t $(IMG_URL)$(call variant-tag,$(VARIANT)) -f Dockerfile.higress . ); )

# HIGRESS_DOCKER_RULE = $(
# 	foreach
# 		VARIANT,
# 		$(DOCKER_BUILD_VARIANTS),
# 		time (
# 			mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ &&
# 			TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ &&
# 			cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) &&
# 			docker build $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) -t $(IMG_URL)$(call variant-tag,$(VARIANT)) -f Dockerfile.higress .
# 		);
# )

HIGRESS_DOCKER_AMD64_RULE ?= $(foreach VARIANT,$(DOCKER_BUILD_VARIANTS), time (mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && TARGET_ARCH=amd64 ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ && cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && docker build $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) --build-arg TARGETARCH=amd64 -t $(IMG_URL)$(call variant-tag,$(VARIANT)) -f Dockerfile.higress . ); )

# HIGRESS_DOCKER_AMD64_RULE = $(
# 	foreach
# 		VARIANT,
# 		$(DOCKER_BUILD_VARIANTS),
# 		time (
# 			mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ &&
# 			TARGET_ARCH=amd64 ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ &&
# 			cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) &&
# 			docker build $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) --build-arg TARGETARCH=amd64 -t $(IMG_URL)$(call variant-tag,$(VARIANT)) -f Dockerfile.higress .
# 		);
# )

