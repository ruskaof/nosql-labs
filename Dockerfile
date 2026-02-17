FROM golang:1.26.0-alpine3.22 AS builder

COPY . .
RUN go build -C ./cmd/app -o /bin/app

FROM scratch
COPY --from=builder /bin/app /bin/app

ENTRYPOINT [ "/bin/app" ]