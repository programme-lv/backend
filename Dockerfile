FROM golang:1.25-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/programme-backend ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /backend

COPY --from=build /out/programme-backend /usr/local/bin/programme-backend
COPY --from=build /src/postgres/migrate ./postgres/migrate

EXPOSE 8080

USER nobody:nogroup

ENTRYPOINT ["programme-backend"]
