VERSION = v0.0.2
LDFLAGS = -ldflags '-s -w'
GOARCH = amd64
linux: export GOOS=linux
darwin: export GOOS=darwin
windows: export GOOS=windows

all: linux darwin windows

linux:
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION} release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels)

darwin:
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION} release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels)

windows:
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION}.exe release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION}.exe release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}.exe
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}.exe
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels.exe terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels.exe)

.PHONY: clean
clean:
	rm -rf release
	rm -f terraform-provider-ssh terraform-provider-ssh.exe terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/terraform-open-ssh-tunnels.exe terraform-provider-ssh_${VERSION} terraform-provider-ssh_${VERSION}.exe
