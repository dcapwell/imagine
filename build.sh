#!/bin/bash

export GOPATH=`pwd`

go get github.com/gorilla/context
go get github.com/keep94/weblogs
go get github.com/nfnt/resize
go get github.com/rcrowley/go-metrics

go build $@
