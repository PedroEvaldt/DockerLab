FROM golang:1.25 AS builder
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o ./shortener ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app/shortener /shortener
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/shortener"]
CMD ["api"]
