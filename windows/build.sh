#!/bin/bash
go build .
wget https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz -O upx.tar.xz
tar -xvf upx.tar.xz
cd upx-3.96-amd64_linux
./upx -o ../bandaid.elf ../bandaid
cd ..
rm -rf bandaid
rm -rf upx.tar.xz
rm -rf upx-3.96-amd64_linux
