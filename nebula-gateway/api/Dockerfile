FROM golang:1.20 as builder
WORKDIR /workspace
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
COPY go.mod ./
COPY cmd ./cmd
RUN go build -o gateway ./cmd/gateway

FROM hyperledger/fabric-tools:2.5
WORKDIR /app
COPY --from=builder /workspace/gateway /usr/local/bin/nebula-gateway
EXPOSE 9000
ENTRYPOINT ["/usr/local/bin/nebula-gateway"]
