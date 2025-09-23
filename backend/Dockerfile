FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o server ./main.go

FROM alpine:3.20
WORKDIR /app
# HTTPS for /img/clean proxy requires CA bundle
RUN apk add --no-cache ca-certificates \
    && update-ca-certificates
# Optional: timezone data for precise Prague time
# RUN apk add --no-cache tzdata
COPY --from=build /app/server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
