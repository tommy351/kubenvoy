# syntax=docker/dockerfile:experimental
FROM golang:1.11

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN --mount=type=cache,target=/root/.cache/go-build go build ./cmd/kds
CMD ["./kds"]
