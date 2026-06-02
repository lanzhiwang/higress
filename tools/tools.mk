tools.bindir = tools/bin
tools.srcdir = tools/src

ifeq ($(origin GOOS), undefined)
	GOOS := $(shell go env GOOS)
endif

# GOOS := linux

ifeq ($(origin GOARCH), undefined)
	GOARCH := $(shell go env GOARCH)
endif

# GOARCH := amd64

# `go get`-able things
# ====================
#
tools/controller-gen = $(tools.bindir)/controller-gen
tools/golangci-lint  = $(tools.bindir)/golangci-lint
tools/kustomize      = $(tools.bindir)/kustomize
tools/kind           = $(tools.bindir)/kind
tools/setup-envtest  = $(tools.bindir)/setup-envtest
$(tools.bindir)/%: $(tools.srcdir)/%/pin.go $(tools.srcdir)/%/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)

# tools/bin/%: tools/src/%/pin.go tools/src/%/go.mod
# #  recipe to execute (from 'tools/tools.mk', line 21):
# 	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run tools/bin/controller-gen
# make[1]: Entering directory '/work'
# cd tools/src/controller-gen && GOOS= GOARCH= go build -o /work/tools/bin/controller-gen $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# make[1]: Leaving directory '/work'
# build-tools:/work#

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run tools/bin/golangci-lint
# make[1]: Entering directory '/work'
# cd tools/src/golangci-lint && GOOS= GOARCH= go build -o /work/tools/bin/golangci-lint $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# make[1]: Leaving directory '/work'
# build-tools:/work#

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run tools/bin/kustomize
# make[1]: Entering directory '/work'
# cd tools/src/kustomize && GOOS= GOARCH= go build -o /work/tools/bin/kustomize $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# make[1]: Leaving directory '/work'
# build-tools:/work#

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run tools/bin/kind
# make[1]: Entering directory '/work'
# cd tools/src/kind && GOOS= GOARCH= go build -o /work/tools/bin/kind $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# make[1]: Leaving directory '/work'
# build-tools:/work#

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run tools/bin/setup-envtest
# make[1]: Entering directory '/work'
# cd tools/src/setup-envtest && GOOS= GOARCH= go build -o /work/tools/bin/setup-envtest $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# make[1]: Leaving directory '/work'
# build-tools:/work#

# `pip install`-able things
# =========================
#
tools/codespell    = $(tools.bindir)/codespell
tools/yamllint     = $(tools.bindir)/yamllint
$(tools.bindir)/%.d/venv: $(tools.srcdir)/%/requirements.txt
	mkdir -p $(@D)
	python3 -m venv $@
	$@/bin/pip3 install -r $< || (rm -rf $@; exit 1)
$(tools.bindir)/%: $(tools.bindir)/%.d/venv
	ln -sf $*.d/venv/bin/$* $@

# tools/bin/%.d/venv: tools/src/%/requirements.txt
# #  recipe to execute (from 'tools/tools.mk', line 29):
# 	mkdir -p $(@D)
# 	python3 -m venv $@
# 	$@/bin/pip3 install -r $< || (rm -rf $@; exit 1)

# tools/bin/%: tools/bin/%.d/venv
# #  recipe to execute (from 'tools/tools.mk', line 33):
# 	ln -sf $*.d/venv/bin/$* $@

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run tools/bin/codespell
# make[1]: Entering directory '/work'
# mkdir -p tools/bin/codespell.d
# python3 -m venv tools/bin/codespell.d/venv
# tools/bin/codespell.d/venv/bin/pip3 install -r tools/src/codespell/requirements.txt || (rm -rf tools/bin/codespell.d/venv; exit 1)
# ln -sf codespell.d/venv/bin/codespell tools/bin/codespell
# rm tools/bin/codespell.d/venv
# make[1]: Leaving directory '/work'
# build-tools:/work#

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run tools/bin/yamllint
# make[1]: Entering directory '/work'
# mkdir -p tools/bin/yamllint.d
# python3 -m venv tools/bin/yamllint.d/venv
# tools/bin/yamllint.d/venv/bin/pip3 install -r tools/src/yamllint/requirements.txt || (rm -rf tools/bin/yamllint.d/venv; exit 1)
# ln -sf yamllint.d/venv/bin/yamllint tools/bin/yamllint
# rm tools/bin/yamllint.d/venv
# make[1]: Leaving directory '/work'
# build-tools:/work#

ifneq ($(GOOS),windows)
# Shellcheck
# ==========
#
tools/shellcheck = $(tools.bindir)/shellcheck
SHELLCHECK_VERSION=0.8.0
SHELLCHECK_ARCH=$(shell uname -m)
# shellcheck uses the same binary on Intel and Apple Silicon Mac.
ifeq ($(GOOS),darwin)
SHELLCHECK_ARCH=x86_64
endif
SHELLCHECK_TXZ = https://github.com/koalaman/shellcheck/releases/download/v$(SHELLCHECK_VERSION)/shellcheck-v$(SHELLCHECK_VERSION).$(GOOS).$(SHELLCHECK_ARCH).tar.xz
tools/bin/$(notdir $(SHELLCHECK_TXZ)):
	mkdir -p $(@D)
	curl -sfL $(SHELLCHECK_TXZ) -o $@
%/bin/shellcheck: %/bin/$(notdir $(SHELLCHECK_TXZ))
	mkdir -p $(@D)
	tar -C $(@D) -Jxmf $< --strip-components=1 shellcheck-v$(SHELLCHECK_VERSION)/shellcheck
endif

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run tools/bin/shellcheck
# make[1]: Entering directory '/work'
# mkdir -p tools/bin
# curl -sfL https://github.com/koalaman/shellcheck/releases/download/v0.8.0/shellcheck-v0.8.0.linux.x86_64.tar.xz -o tools/bin/shellcheck-v0.8.0.linux.x86_64.tar.xz
# mkdir -p tools/bin
# tar -C tools/bin -Jxmf tools/bin/shellcheck-v0.8.0.linux.x86_64.tar.xz --strip-components=1 shellcheck-v0.8.0/shellcheck
# make[1]: Leaving directory '/work'
# build-tools:/work#
