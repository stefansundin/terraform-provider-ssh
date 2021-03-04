VERSION = v0.1.0
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

proto:
	protoc -I ssh/proto/ ssh/proto/tunnel.proto --go_out=plugins=grpc:ssh/proto

linux: proto
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip
	go build -o release/terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip release/terraform-provider-ssh_${VERSION}

linux_arm: proto
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip
	go build -o release/terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip release/terraform-provider-ssh_${VERSION}

linux_arm64: proto
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip
	go build -o release/terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip release/terraform-provider-ssh_${VERSION}

darwin: proto
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION} release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip
	go build -o release/terraform-provider-ssh_${VERSION}
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip release/terraform-provider-ssh_${VERSION}

windows: proto
	mkdir -p release
	rm -f terraform-provider-ssh_${VERSION}.exe release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip
	go build -o release/terraform-provider-ssh_${VERSION}.exe
	zip release/terraform-provider-ssh_${VERSION}_${GOOS}_${GOARCH}.zip release/terraform-provider-ssh_${VERSION}.exe

clean:
	rm -rf release ssh/proto/*.go
