FROM golang:1.27-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cache-db .

FROM alpine:3.19
RUN adduser -D -u 10001 appuser
COPY --from=build /out/cache-db /usr/local/bin/cache-db
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=2s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["cache-db"]
