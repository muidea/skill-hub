FROM golang:1.24 AS builder

WORKDIR /src
COPY . .

RUN go build -o /out/skill-hub ./application/skill-hub/cmd

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /out/skill-hub /usr/local/bin/skill-hub
COPY application/skill-hub/docker/bootstrap.sh /bootstrap.sh

ENTRYPOINT ["/bootstrap.sh"]
