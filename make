#!/bin/bash

ROOT="`pwd`/build"
BIN="$ROOT/usr/local/bin"
RPMDIR="/root/rpmbuild/RPMS/x86_64"

if [ ! -x "$ROOT" ]; then
    echo $ROOT not exists
    exit 1
fi

if [ ! -x "$BIN" ]; then
    mkdir -p $BIN
fi

go build -ldflags "-s -w" -a -tags netgo -installsuffix netgo
mv eru-agent $BIN
OUTPUT=($($BIN/eru-agent -v 2>&1))
VERSION=${OUTPUT[-1]}
echo $VERSION build begin

fpm -f -s dir -t rpm -n eru-agent --epoch 0 -v $VERSION --iteration 1.el7 -C $ROOT -p $RPMDIR --verbose --category 'Development/App' --description 'docker eru agent' --url 'github.com' --license 'BSD'  --no-rpm-sign usr etc

