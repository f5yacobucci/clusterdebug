FROM golang

COPY clusterdebug /

ENTRYPOINT ["/clusterdebug"]
