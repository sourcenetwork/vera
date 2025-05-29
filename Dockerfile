FROM golang:1.23.8 as builder

WORKDIR /app

# Cache deps
COPY go.* /app/
RUN go mod download

# Build
COPY . /app
RUN --mount=type=cache,target=/root/.cache go build -o /build/sourcehubd ./cmd/sourcehubd

# Deployment entrypoint
FROM debian:bookworm-slim

COPY --from=builder /app/build/sourcehubd /usr/local/bin/sourcehubd

RUN useradd --create-home --home-dir /home/node node && mkdir /sourcehub && chown node:node /sourcehub && ln -s /sourcehub /home/node/.sourcehub

USER node
VOLUME ["/sourcehub"]
ENTRYPOINT ["sourcehubd"]
CMD ["start"]
