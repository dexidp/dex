#!/bin/bash
rm -rf dist
npm i
npm run build
cp assets/img/favicon.ico dist
# can't use -o option
go-bindata-assetfs -pkg ui dist/...
mv bindata_assetfs.go ../../ui/
