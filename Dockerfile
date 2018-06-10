FROM golang:1.10-alpine3.7 AS build

WORKDIR $GOPATH/src/github.com/errm/ekstrap
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux \
    go build \
    -o /ekstrap .

FROM alpine:3.7
WORKDIR /app
COPY --from=build /ekstrap /app/
ENTRYPOINT ["./ekstrap"]
