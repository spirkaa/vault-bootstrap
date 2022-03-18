## Build
FROM golang:1.17-alpine AS build

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /vault-bootstrap ./cmd/vault-bootstrap/main.go

## Deploy
FROM scratch

WORKDIR /

COPY --from=build /vault-bootstrap .

USER 1001

CMD ["/vault-bootstrap"]
