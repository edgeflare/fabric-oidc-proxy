FROM golang:1.23-alpine3.20 as BUILDER
RUN apk add --no-cache git
WORKDIR /workspace
COPY . .
RUN go mod tidy
ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} \
  go build -ldflags='-w -s -extldflags "-static"' -a -o fabric-oidc-proxy .

# Copy binary into final (alpine) image
FROM alpine:3.20
RUN adduser -D -h /workspace 1000
USER 1000
WORKDIR /workspace
COPY --from=BUILDER /workspace/fabric-oidc-proxy .
EXPOSE 8080
ENTRYPOINT ["/workspace/fabric-oidc-proxy"]
CMD ["start"]
