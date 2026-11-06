#!/bin/bash
set -e

go build -tags='excl_mysql,excl_consul' .
install -m 755 pocket /usr/local/bin/pocket