#docker build --rm --build-arg APP_ROOT=/go/src/cluster-autoscaler -t cluster-autoscaler:latest -f Dockerfile .
#0 ----------------------------
FROM golang:1.17.4
ARG  APP_ROOT
WORKDIR ${APP_ROOT}
COPY ./ ${APP_ROOT}

# install upx
RUN sed -i "s/deb.debian.org/mirrors.aliyun.com/g" /etc/apt/sources.list \
  && sed -i "s/security.debian.org/mirrors.aliyun.com/g" /etc/apt/sources.list \
  && apt-get update \
  && apt-get install upx musl-dev git -y

# build code
RUN go mod tidy \
  && CGO_ENABLED=0 GOOS=linux go build -ldflags \
  "-w -s" -o main \
  && strip --strip-unneeded main \
  && upx --lzma main

#1 ----------------------------
FROM alpine:latest
ARG APP_ROOT
WORKDIR /app
COPY --from=0 ${APP_ROOT}/main .
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
  && apk add --no-cache openssh jq curl busybox-extras \
  && rm -rf /var/cache/apk/*

ENTRYPOINT ["/app/main"]
