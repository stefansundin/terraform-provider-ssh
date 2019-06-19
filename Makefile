VERSION = v0.0.3
LDFLAGS = -ldflags '-s -w' -gcflags=-trimpath=${PWD} -asmflags=-trimpath=${PWD}
GOARCH = amd64
linux: export GOOS=linux
linux_arm: export GOOS=linux
linux_arm: export GOARCH=arm
linux_arm: export GOARM=6
linux_arm64: export GOOS=linux
linux_arm64: export GOARCH=arm64
darwin: export GOOS=darwin
windows: export GOOS=windows

.PHONY: all linux linux_arm linux_arm64 darwin windows clean

all: linux linux_arm linux_arm64 darwin windows

linux:
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION} release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels)

linux_arm:
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION} release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels)

linux_arm64:
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

clean:
	rm -rf release
	rm -f terraform-provider-ssh terraform-provider-ssh.exe terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/terraform-open-ssh-tunnels.exe terraform-provider-ssh_${VERSION} terraform-provider-ssh_${VERSION}.exe
