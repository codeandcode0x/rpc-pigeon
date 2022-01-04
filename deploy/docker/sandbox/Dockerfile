FROM golang:stretch  as testing

WORKDIR /opt/app/

RUN apt-get update -y && apt-get install git vim curl wget fish lrzsz -y && \
    go env -w GOPROXY=https://goproxy.cn,direct