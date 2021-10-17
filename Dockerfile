FROM golang:latest as builder

WORKDIR /app
COPY . /app/.
RUN CGO_ENABLED=0 go build -o prometheus2mqtt .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/prometheus2mqtt .
CMD ["./prometheus2mqtt"]
