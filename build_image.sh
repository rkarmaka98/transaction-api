#!/bin/bash

docker build -t tx-api .
docker run -p 8080:8080 tx-api
