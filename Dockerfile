FROM golang:1.25-alpine AS build
WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o api ./cmd/api

FROM alpine:latest
COPY --from=build /usr/src/app/api /app
EXPOSE 8080
CMD [ "/app" ]