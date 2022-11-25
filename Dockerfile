FROM scratch

WORKDIR /

COPY build/vault-bootstrap .

USER 1001

CMD ["/vault-bootstrap"]
