FROM golang:1.14-alpine AS build
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.io,direct && \
    go env -w GOSUMDB=sum.golang.google.cn
WORKDIR /go/src/containerd
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -v -o /containerd cmd/containerd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /containerd /containerd
RUN chmod +x ./containerd
CMD ["/containerd"] 
