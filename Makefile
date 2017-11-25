VERSION = 0.0.1
LDFLAGS = -ldflags '-s -w'
GOARCH = amd64
linux: export GOOS=linux
darwin: export GOOS=darwin
windows: export GOOS=windows

all: linux darwin windows

linux:
	go build $(LDFLAGS)
	mkdir -p release
	rm -f release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.xz
	xz terraform-provider-ssh
	mv terraform-provider-ssh.xz release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.xz

darwin:
	go build $(LDFLAGS)
	mkdir -p release
	rm -f release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.xz
	xz terraform-provider-ssh
	mv terraform-provider-ssh.xz release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.xz

windows:
	go build $(LDFLAGS)
	mkdir -p release
	rm -f release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.xz
	xz terraform-provider-ssh.exe
	mv terraform-provider-ssh.exe.xz release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.exe.xz

.PHONY: clean
clean:
	rm -rf release
	rm -f terraform-provider-ssh terraform-provider-ssh.exe
