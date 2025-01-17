FROM golang:1.17
ARG FN
ENV CGO_ENABLED=0
WORKDIR /go/src/
COPY . .
RUN go build -v -o /usr/local/bin/function ./${FN}

FROM alpine:3.15.0
COPY --from=0 /usr/local/bin/function /usr/local/bin/function
CMD ["function"]
