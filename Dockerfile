FROM node:24-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.27-alpine AS build
WORKDIR /src
ARG VERSION=dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN rm -rf internal/ui/dist && mkdir -p internal/ui/dist
COPY --from=web /web/dist ./internal/ui/dist
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X goread/internal/version.Version=${VERSION}" -o /goread ./cmd/app

FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /goread /usr/local/bin/goread
EXPOSE 8080
ENTRYPOINT ["goread"]
