FROM golang:1.16
ADD . /opt/otic-test-app/
WORKDIR /opt/otic-test-app
RUN go build -o /bin/pause main.go
RUN go test -c ./test/kpm -o /bin/kpm.test
RUN go test -c ./test/rc -o /bin/rc.test