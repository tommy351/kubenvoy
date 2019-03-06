FROM golang:1.11-alpine AS base

RUN apk add --update --no-cache git ca-certificates

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

ENV CGO_ENABLED=0
COPY . ./
RUN go build -o /usr/local/bin/kds -tags netgo -ldflags "-w" ./cmd/kds
CMD ["kds"]

FROM scratch

COPY --from=base /usr/local/bin/kds /usr/local/bin/kds
CMD ["kds"]
