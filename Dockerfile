FROM golang:1.26-alpine AS build
WORKDIR /src

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

# download deps first — cached unless go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/blog ./cmd/web
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/blog-cli ./cmd/blog

FROM alpine:3.20
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY --from=build /bin/blog /app/blog
COPY --from=build /bin/blog-cli /app/blog-cli
EXPOSE 2026
ENTRYPOINT ["/app/blog"]
