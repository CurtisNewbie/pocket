#!/bin/bash
set -e

go build -tags='excl_mysql,excl_consul' .
go install -tags='excl_mysql,excl_consul' .
