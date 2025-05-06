FROM golang:1.23.8 AS builder

WORKDIR /app

# Cache deps
COPY go.* /app/

RUN go mod download

# Build
COPY . /app
RUN --mount=type=cache,target=/root/.cache make build
RUN --mount=type=cache,target=/root/.cache go build -o build/tx_err_forwader cmd/tx_err_forwader

# Deployment entrypoint
FROM debian:bookworm-slim

RUN useradd --create-home --home-dir /home/node node && mkdir /sourcehub && chown node:node /sourcehub && ln -s /sourcehub /home/node/.sourcehub
USER node
VOLUME ["/sourcehub"]

COPY --from=builder /app/build/sourcehubd /usr/local/bin/sourcehubd
COPY --from=builder /app/build/tx_err_forwader /usr/local/bin/tx_err_forwader

ENTRYPOINT ["sourcehubd"]
CMD ["start"]
