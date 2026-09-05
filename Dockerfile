FROM golang:1.26-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go tool templ generate templates/*.templ
RUN go build -o notes-viewer .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=build /app/assets /app/assets
COPY --from=build /app/notes-viewer /app/notes-viewer
COPY --from=build /app/templates /app/templates

EXPOSE 8080
ENTRYPOINT ["/app/notes-viewer"]
