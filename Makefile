OUTDIR := build

.PHONY: all lint security clean icons generate \
        build-windows build-darwin build-linux

all: fmt vet build-windows build-darwin build-linux

# --- Code quality ---

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet golangci-lint vulncheck

golangci-lint:
	golangci-lint run ./...

vulncheck:
	govulncheck ./...

security: golangci-lint vulncheck

# --- Asset generation ---

icons:
	go run ./cmd/icongen/

generate:
	go generate ./cmd/micmonitor-tray/

# --- Windows (amd64) ---

build-windows: $(OUTDIR)/windows
	GOOS=windows GOARCH=amd64 go build -o $(OUTDIR)/windows/miccheck.exe ./cmd/miccheck/
	GOOS=windows GOARCH=amd64 go build -o $(OUTDIR)/windows/micmonitor-cli.exe ./cmd/micmonitor-cli/
	GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o $(OUTDIR)/windows/micmonitor-tray.exe ./cmd/micmonitor-tray/

# --- macOS (arm64 + amd64) ---

build-darwin: build-darwin-arm64 build-darwin-amd64

build-darwin-arm64: $(OUTDIR)/darwin-arm64
	GOOS=darwin GOARCH=arm64 go build -o $(OUTDIR)/darwin-arm64/miccheck ./cmd/miccheck/
	GOOS=darwin GOARCH=arm64 go build -o $(OUTDIR)/darwin-arm64/micmonitor-cli ./cmd/micmonitor-cli/

build-darwin-amd64: $(OUTDIR)/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build -o $(OUTDIR)/darwin-amd64/miccheck ./cmd/miccheck/
	GOOS=darwin GOARCH=amd64 go build -o $(OUTDIR)/darwin-amd64/micmonitor-cli ./cmd/micmonitor-cli/

# --- Linux (amd64) ---

build-linux: $(OUTDIR)/linux
	GOOS=linux GOARCH=amd64 go build -o $(OUTDIR)/linux/miccheck ./cmd/miccheck/
	GOOS=linux GOARCH=amd64 go build -o $(OUTDIR)/linux/micmonitor-cli ./cmd/micmonitor-cli/

# --- Output directories ---

$(OUTDIR)/windows $(OUTDIR)/darwin-arm64 $(OUTDIR)/darwin-amd64 $(OUTDIR)/linux:
	mkdir -p $@

# --- Cleanup ---

clean:
	rm -rf $(OUTDIR)
