FROM docker.m.daocloud.io/library/golang:1.23.7 as builder
# 后期需要优化指定运行的用户和用户组
WORKDIR /app

COPY .. .
# 构建
RUN CGO_ENABLED=0 go build -o ingress-manager main.go

FROM docker.m.daocloud.io/library/alpine:3.15.3

WORKDIR /app

COPY --from=builder /app/ingress-manager .

CMD ["./ingress-manager"]
# docker build -t dragonzw/ingreminikube mount 11:/home/docker/client-goss-manager:1.0.0 .