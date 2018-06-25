FROM alpine:3.7
COPY ekstrap /
ENTRYPOINT ["./ekstrap"]
