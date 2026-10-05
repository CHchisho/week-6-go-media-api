FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /week-6-go-media-api .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /week-6-go-media-api .
COPY openapi.yaml .
EXPOSE 8080
CMD ["./week-6-go-media-api"]
