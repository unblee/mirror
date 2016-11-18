FROM scratch
COPY mirror /
ENTRYPOINT ["/mirror"]