FROM golang:1.23.4-alpine3.20 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/devinspector ./cmd/app

FROM alpine:3.20
RUN addgroup -S devinspector && adduser -S devinspector -G devinspector
WORKDIR /workspace
COPY --from=build /out/devinspector /usr/local/bin/devinspector
USER devinspector
HEALTHCHECK CMD devinspector version || exit 1
ENTRYPOINT ["devinspector"]
CMD ["scan", "."]
