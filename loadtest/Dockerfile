FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -o /out/loadtest ./cmd/loadtest

RUN go install github.com/tsenart/vegeta/v12@v12.12.0

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
  && adduser -D -H -u 10001 app
USER app
WORKDIR /app

COPY --from=builder /out/loadtest /usr/local/bin/loadtest
COPY --from=builder /go/bin/vegeta /usr/local/bin/vegeta

ENTRYPOINT ["/usr/local/bin/loadtest"]