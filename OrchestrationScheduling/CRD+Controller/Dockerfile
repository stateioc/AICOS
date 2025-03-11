FROM golang:1.20 AS builder
WORKDIR /go/src/box-controller
COPY . .
ENV GOPROXY https://goproxy.cn,direct
ENV GOSUMDB=off CGO_ENABLED=0 GOOS=linux

# RUN go version
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o box-controller .

FROM frolvlad/alpine-glibc:alpine-3.10_glibc-2.30
ENV TZ Asia/Shanghai
WORKDIR /app
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /go/src/box-controller/box-controller .
CMD ["./box-controller"]

