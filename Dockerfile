FROM golang
WORKDIR /
COPY runoverworkflows.go /runoverworkflows.go
RUN go build -o /entrypoint
CMD /entrypoint
