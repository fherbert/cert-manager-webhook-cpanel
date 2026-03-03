FROM golang:1.21-alpine AS builder

WORKDIR /workspace

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o webhook .

FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /workspace/webhook .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER 65532:65532

ENTRYPOINT ["/webhook"]
