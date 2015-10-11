#!/bin/bash

browser-sync start --config bs-config.js > /dev/null 2>&1 &

autoexec -sve "killall bullettime ; GO15VENDOREXPERIMENT=1 go build github.com/matrix-org/bullettime && ./bullettime 8009 & browser-sync reload --port 3080" main.go */*.go */*/*.go */*/*/*.go */*/*/*/*.go

