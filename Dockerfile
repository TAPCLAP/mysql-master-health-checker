# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mysql-master-health-checker ./cmd/mysql-master-health-checker

FROM gcr.io/distroless/static-debian12:nonroot
ENV METRICS_LISTEN_ADDR=0.0.0.0:18849
COPY --from=builder /out/mysql-master-health-checker /mysql-master-health-checker
USER nonroot:nonroot
EXPOSE 18848 18849
ENTRYPOINT ["/mysql-master-health-checker"]
