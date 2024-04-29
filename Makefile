SHELL=/bin/sh

OUTPUT=tgmsgdel

.PHONY: clean cleanall cleanfiles debug run


export CGO_LDFLAGS="-L/usr/local/opt/openssl/lib"

$(OUTPUT): main.go
	go build -o $@

debug:
	dlv debug .

run:
	go run .

clean:
	-rm $(OUTPUT)

test:
	go test ./... -race -cover

fuzz:
	go test -fuzz=Fuzz -fuzztime 30s ./internal/secure
	go test -fuzz=Fuzz -fuzztime 30s ./internal/mtp

cleanfiles:
	-rm -rf tdlib-db tdlib-files

dist:
	goreleaser check
	goreleaser release --snapshot --rm-dist -p 2
.PHONY: dist

cleanall: clean cleanfiles
