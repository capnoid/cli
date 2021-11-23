FROM alpine:3.10
COPY ./airplane /bin
ENTRYPOINT ["/bin/airplane"]
