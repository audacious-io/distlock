FROM alpine

COPY bin/lockerd-linux-amd64 /lockerd

ENTRYPOINT ["/lockerd"]
CMD ["server"]
EXPOSE 12000/tcp
