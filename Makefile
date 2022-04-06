VERSION = v0.2.0
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

.PHONY: all pb linux linux_arm linux_arm64 darwin windows clean

all: linux linux_arm linux_arm64 darwin windows

pb:
	protoc -I pb/ pb/ssh_tunnel.proto --go_out=plugins=grpc:pb

linux: pb
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION} release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels)

linux_arm: pb
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION} release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels)

linux_arm64: pb
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION} release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels)

darwin: pb
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION} release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels)

windows: pb
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION}.exe release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels_${VERSION}.exe release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip
	go build $(LDFLAGS) -o terraform-provider-ssh_${VERSION}.exe
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip terraform-provider-ssh_${VERSION}.exe
	go build $(LDFLAGS) -o terraform-open-ssh-tunnels/terraform-open-ssh-tunnels.exe terraform-open-ssh-tunnels/main.go
	(cd terraform-open-ssh-tunnels && zip ../release/terraform-open-ssh-tunnels_${VERSION}_${GOOS}_${GOARCH}.zip terraform-open-ssh-tunnels.exe)

clean:
	rm -rf release
	rm -f terraform-provider-ssh terraform-provider-ssh.exe terraform-open-ssh-tunnels/terraform-open-ssh-tunnels terraform-open-ssh-tunnels/terraform-open-ssh-tunnels.exe terraform-provider-ssh_${VERSION} terraform-provider-ssh_${VERSION}.exe pb/ssh_tunnel.pb.go

install: darwin
	mkdir -p ~/.terraform.d/plugins/darwin_amd64
	cp terraform-provider-ssh_${VERSION} ~/.terraform.d/plugins/darwin_amd64
