FROM golang:1.11

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
CMD ["go", "run", "./cmd/kds"]
