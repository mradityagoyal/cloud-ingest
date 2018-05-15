FROM golang:1.9

WORKDIR /go/src/github.com/GoogleCloudPlatform/cloud-ingest
COPY . .

RUN go get -d cloud.google.com/go/pubsub
RUN go get -d cloud.google.com/go/spanner
RUN go get -d github.com/golang/glog
RUN go get -d github.com/golang/groupcache/lru
RUN go get -d github.com/golang/mock/gomock
RUN go get -d github.com/golang/mock/mockgen
RUN go get -d github.com/golang/protobuf/protoc-gen-go
RUN go get -d golang.org/x/time/rate

RUN go install -v ./agent/...

ENTRYPOINT ["agentmain"]
