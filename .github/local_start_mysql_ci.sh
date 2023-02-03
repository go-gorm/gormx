#!/bin/bash

set -e

docker run \
  -d \
  -p 9910:3306 \
  -e MYSQL_DATABASE=gorm \
  -e MYSQL_USER=gorm \
  -e MYSQL_PASSWORD=gorm \
  -e MYSQL_ALLOW_EMPTY_PASSWORD=yes \
  mysql:latest