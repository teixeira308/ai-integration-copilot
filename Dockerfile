FROM golang:1.25.1-alpine AS builder
WORKDIR /workspace

COPY backend/go.mod ./
COPY backend ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /workspace/bin/server ./cmd/server

FROM gcr.io/distroless/base-debian11
COPY --from=builder /workspace/bin/server /bin/server

EXPOSE 8080

ENTRYPOINT ["/bin/server"]
