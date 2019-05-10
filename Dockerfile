FROM golang:1.9.6 as builder

COPY ./ /go/src/github.com/contiv/auth_proxy

WORKDIR /go/src/github.com/contiv/auth_proxy

RUN VERSION=$(git describe --tags --always) ./scripts/build_in_container.sh

#FROM scratch
FROM ubuntu:16.04 

COPY ./ui/app /ui
COPY --from=builder /go/src/github.com/contiv/auth_proxy/build/output/auth_proxy /auth_proxy

WORKDIR /

ENTRYPOINT ["./auth_proxy"]
