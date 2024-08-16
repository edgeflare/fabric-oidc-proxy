FROM golang:1.23rc1 AS builder
WORKDIR /fabric/chaincode
COPY . .
RUN go mod tidy
RUN mkdir -p bin
RUN go build -ldflags='-w -s -extldflags "-static"' -a -o ./bin/ ./example-ccaas/...

# runtime image
FROM debian:bookworm
RUN adduser -D -h /fabric/chaincode 1000
COPY --from=builder /fabric/chaincode/bin/example-ccaas /fabric/chaincode/assetcc
RUN chmod +x /fabric/chaincode/assetcc
USER 1000
WORKDIR /fabric/chaincode
ENV CHAINCODE_NAME=assetcc
ENV CHAINCODE_VERSION=1.0
# Obtain CHAINCODE_ID by running `peer lifecycle chaincode queryinstalled` on the peer using admin credentials
# eg CHAINCODE_ID=${CHAINCODE_NAME}_${CHAINCODE_VERSION}:0262396ccaffaa2174bc09f750f742319c4f14d60b16334d2c8921b6842c090c
ENV CHAINCODE_ID=${CHAINCODE_NAME}_${CHAINCODE_VERSION}:"sha256sum packaged_cc.tar.gz"
ENV CHAINCODE_TLS_DISABLED=true
ENV CHAINCODE_TLS_KEY=/fabric/chaincode/tls/server.key
ENV CHAINCODE_TLS_CERT=/fabric/chaincode/tls/server.crt
ENV CHAINCODE_TLS_CLIENT_CACERT=""
ENV CHAINCODE_SERVER_ADDRESS=0.0.0.0:7052
EXPOSE 7052
ENTRYPOINT ["/fabric/chaincode/assetcc"]
