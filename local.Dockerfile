## Build
FROM golang:1.20-alpine AS build

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /vault-bootstrap

## Deploy
FROM scratch

LABEL org.opencontainers.image.description="Init and unseal Hashicorp Vault on Kubernetes"

WORKDIR /

COPY --from=build /vault-bootstrap .

USER 1001

CMD ["/vault-bootstrap"]
