FROM golang
WORKDIR /
ENV GO111MODULE=auto
COPY runoverworkflows.go /runoverworkflows.go
RUN go build -o /entrypoint
CMD /entrypoint
