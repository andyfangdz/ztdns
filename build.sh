#!/usr/bin/env bash

docker build -t gcr.io/andyfang-biz-prod/ztdns:latest .
docker push gcr.io/andyfang-biz-prod/ztdns:latest
