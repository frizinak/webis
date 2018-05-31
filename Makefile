SRC := $(shell find . -type f -name '*.go')
CROSSARCH := amd64 386
CROSSOS := darwin linux openbsd netbsd freebsd windows
CROSS_SERVER := $(foreach os,$(CROSSOS),$(foreach arch,$(CROSSARCH),dist/webis.$(os).$(arch)))
CROSS_CLI := $(foreach os,$(CROSSOS),$(foreach arch,$(CROSSARCH),dist/webis-cli.$(os).$(arch)))

.PHONY: reset test bench build cross cross-server cross-cli

test:
	go test -count=1 \
		$(shell find . -maxdepth 3 -type f -name '*_test.go' -exec dirname {} \;)

bench:
	go test -test.benchmem -bench=. \
		$(shell find . -maxdepth 3 -type f -name '*_test.go' -exec dirname {} \;)

build: dist/webis dist/webis-cli

dist/webis: $(SRC)
	@- mkdir dist 2>/dev/null
	go build -o dist/webis ./cmd/webis/*.go

dist/webis-cli: $(SRC)
	@- mkdir dist 2>/dev/null
	go build -o dist/webis-cli ./cmd/webis-cli/*.go

install:
	go install github.com/frizinak/webis/cmd/webis
	go install github.com/frizinak/webis/cmd/webis-cli

cross: cross-server cross-cli
cross-server: $(CROSS_SERVER)
cross-cli: $(CROSS_CLI)

$(CROSS_SERVER): $(SRC)
	@- mkdir dist 2>/dev/null
	gox \
		-osarch="$(shell echo "$@" | cut -d'/' -f2- | cut -d'.' -f2- | sed 's/\./\//')" \
		-output="dist/webis.{{.OS}}.{{.Arch}}" \
		./cmd/webis/
	if [ -f "$@.exe" ]; then mv "$@.exe" "$@"; fi

$(CROSS_CLI): $(SRC)
	@- mkdir dist 2>/dev/null
	gox \
		-osarch="$(shell echo "$@" | cut -d'/' -f2- | cut -d'.' -f2- | sed 's/\./\//')" \
		-output="dist/webis-cli.{{.OS}}.{{.Arch}}" \
		./cmd/webis-cli/
	if [ -f "$@.exe" ]; then mv "$@.exe" "$@"; fi

reset:
	-rm -rf dist

