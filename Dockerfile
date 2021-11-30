FROM alpine:3.15
COPY ./airplane /bin
ENTRYPOINT ["/bin/airplane"]
