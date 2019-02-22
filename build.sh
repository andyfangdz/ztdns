#!/usr/bin/env bash
set -ex

docker build --no-cache -t gcr.io/andyfang-biz-prod/ztdns:latest .
docker push gcr.io/andyfang-biz-prod/ztdns:latest
