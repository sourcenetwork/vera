FROM golang:1.23.8 as builder

ARG IGNITE_VERSION="28.10.0"

# Ignite CLI prompts users for tracking info which hangs the tty
# See https://github.com/ignite/cli/blob/main/ignite/internal/analytics/analytics.go#L71
ENV DO_NOT_TRACK=1

# Set DEV="1" to install ignite cli for the dev container
ARG DEV=0

# Setup env and dev tools
RUN apt update &&\
    apt-get install --yes git make curl jq; \
    if [ "$DEV" = "1" ]; then\
        mkdir /env &&\
        cd /env &&\
        curl -L https://github.com/ignite/cli/releases/download/v${IGNITE_VERSION}/ignite_${IGNITE_VERSION}_linux_amd64.tar.gz > ignite.tar.gz &&\
        tar xvf ignite.tar.gz &&\
        mv ignite /usr/local/bin/ignite &&\
        cd / && rm -rf /env;\
    fi

WORKDIR /app

# Cache deps
COPY go.* /app/

RUN go mod download

# Build
COPY . /app
RUN --mount=type=cache,target=/root/.cache make build

# Dev image entrypoint
ENTRYPOINT ["scripts/dev-entrypoint.sh"]
CMD ["start"]


# Deployment entrypoint
FROM debian:bookworm-slim

RUN useradd -ms /bin/bash node
USER node

COPY --from=builder /app/build/sourcehubd /usr/local/bin/sourcehubd

ENTRYPOINT ["sourcehubd"]
CMD ["start"]