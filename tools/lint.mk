# This is a wrapper to do lint checks
#
# All make targets related to lint are defined in this file.

##@ Lint

GITHUB_ACTION ?=

.PHONY: lint
lint: ## Run all linter of code sources, including golint, yamllint, whitenoise lint and codespell.

# lint-deps is run separately in CI to separate the tooling install logs from the actual output logs generated
# by the lint tooling.
.PHONY: lint-deps
lint-deps: ## Everything necessary to lint

GOLANGCI_LINT_FLAGS ?= $(if $(GITHUB_ACTION),--out-format=github-actions)
.PHONY: lint.golint
lint: lint.golint
lint-deps: $(tools/golangci-lint)
lint.golint: $(tools/golangci-lint)
	$(tools/golangci-lint) run $(GOLANGCI_LINT_FLAGS) --config=tools/linter/golangci-lint/.golangci.yml

.PHONY: lint.yamllint
lint: lint.yamllint
lint-deps: $(tools/yamllint)
lint.yamllint: $(tools/yamllint)
	$(tools/yamllint) --config-file=tools/linter/yamllint/.yamllint $$(git ls-files :*.yml :*.yaml | xargs -L1 dirname | sort -u)

CODESPELL_FLAGS ?= $(if $(GITHUB_ACTION),--disable-colors)
.PHONY: lint.codespell
lint: lint.codespell
lint-deps: $(tools/codespell)
lint.codespell: CODESPELL_SKIP := $(shell cat tools/linter/codespell/.codespell.skip | tr \\n ',')
lint.codespell: $(tools/codespell)

# This ::add-matcher/::remove-matcher business is based on
# https://github.com/codespell-project/actions-codespell/blob/2292753ad350451611cafcbabc3abe387491339a/entrypoint.sh
# We do this here instead of just using
# codespell-project/codespell-problem-matcher@v1 so that the matcher
# doesn't apply to the other linters that `make lint` also runs.
#
# This recipe is written a little awkwardly with everything running in
# one shell, this is because we want the ::remove-matcher lines to get
# printed whether or not it finds complaints.
	@PS4=; set -e; { \
	  if test -n "$$GITHUB_ACTION"; then \
	    printf '::add-matcher::$(CURDIR)/tools/linter/codespell/matcher.json\n'; \
	    trap "printf '::remove-matcher owner=codespell-matcher-default::\n::remove-matcher owner=codespell-matcher-specified::\n'" EXIT; \
	  fi; \
	  (set -x; $(tools/codespell) $(CODESPELL_FLAGS) --skip $(CODESPELL_SKIP) --ignore-words tools/linter/codespell/.codespell.ignorewords --check-filenames --check-hidden -q2); \
	}

.PHONY: lint.shellcheck
lint: lint.shellcheck
lint-deps: $(tools/shellcheck)
lint.shellcheck: $(tools/shellcheck)
	$(tools/shellcheck) tools/hack/*.sh

# build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run lint
# make[1]: Entering directory '/work'
# cd tools/src/golangci-lint && GOOS= GOARCH= go build -o /work/tools/bin/golangci-lint $(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# tools/bin/golangci-lint run  --config=tools/linter/golangci-lint/.golangci.yml
# mkdir -p tools/bin/yamllint.d
# python3 -m venv tools/bin/yamllint.d/venv
# tools/bin/yamllint.d/venv/bin/pip3 install -r tools/src/yamllint/requirements.txt || (rm -rf tools/bin/yamllint.d/venv; exit 1)
# ln -sf yamllint.d/venv/bin/yamllint tools/bin/yamllint
# tools/bin/yamllint --config-file=tools/linter/yamllint/.yamllint $(git ls-files :*.yml :*.yaml | xargs -L1 dirname | sort -u)
# mkdir -p tools/bin/codespell.d
# python3 -m venv tools/bin/codespell.d/venv
# tools/bin/codespell.d/venv/bin/pip3 install -r tools/src/codespell/requirements.txt || (rm -rf tools/bin/codespell.d/venv; exit 1)
# ln -sf codespell.d/venv/bin/codespell tools/bin/codespell
# PS4=; set -e; { \
#   if test -n "$GITHUB_ACTION"; then \
#     printf '::add-matcher::/root/huzhi/higress/tools/linter/codespell/matcher.json\n'; \
#     trap "printf '::remove-matcher owner=codespell-matcher-default::\n::remove-matcher owner=codespell-matcher-specified::\n'" EXIT; \
#   fi; \
#   (set -x; tools/bin/codespell  --skip .git,.idea,*.png,*.woff,*.woff2,*.eot,*.ttf,*.jpg,*.ico,*.svg,./docs/html/*,go.mod,go.sum,bin, --ignore-words tools/linter/codespell/.codespell.ignorewords --check-filenames --check-hidden -q2); \
# }
# mkdir -p tools/bin
# curl -sfL https://github.com/koalaman/shellcheck/releases/download/v0.8.0/shellcheck-v0.8.0.linux.x86_64.tar.xz -o tools/bin/shellcheck-v0.8.0.linux.x86_64.tar.xz
# mkdir -p tools/bin
# tar -C tools/bin -Jxmf tools/bin/shellcheck-v0.8.0.linux.x86_64.tar.xz --strip-components=1 shellcheck-v0.8.0/shellcheck
# tools/bin/shellcheck tools/hack/*.sh
# rm tools/bin/codespell.d/venv tools/bin/yamllint.d/venv
# make[1]: Leaving directory '/work'
# build-tools:/work#
