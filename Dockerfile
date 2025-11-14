FROM golang:1.24.7 AS builder

WORKDIR /app

# Cache deps
COPY go.* /app/
RUN go mod download

# Build
COPY . /app
RUN --mount=type=cache,target=/root/.cache go build -o /app/build/sourcehubd ./cmd/sourcehubd

# Deployment entrypoint
FROM debian:bookworm-slim

COPY scripts/entrypoint.sh /usr/local/bin/entrypoint.sh
COPY --from=builder /app/build/sourcehubd /usr/local/bin/sourcehubd

RUN useradd --create-home --home-dir /home/node node && mkdir /sourcehub && chown node:node /sourcehub && ln -s /sourcehub /home/node/.sourcehub && chown node:node -R /home/node

USER node
VOLUME ["/sourcehub"]
ENTRYPOINT ["entrypoint.sh"]
CMD ["sourcehubd", "start"]
