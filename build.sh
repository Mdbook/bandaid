#!/bin/bash
# Build the binary
go build .
# Get UPX and unzip
wget https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz -O upx.tar.xz
tar -xvf upx.tar.xz
cd upx-3.96-amd64_linux
# Pack the binary
./upx -o ../bandaid.elf ../bandaid
# Clean up
cd ..
rm -rf bandaid
rm -rf upx.tar.xz
rm -rf upx-3.96-amd64_linux
