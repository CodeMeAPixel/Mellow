FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/mellow ./cmd/mellow

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 mellow
USER mellow
COPY --from=build /out/mellow /usr/local/bin/mellow
ENTRYPOINT ["/usr/local/bin/mellow"]
