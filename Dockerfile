FROM golang:alpine AS build-env
RUN apk add --no-cache --virtual --update glide git make

ADD . /go/src/github.com/CyrilPeponnet/slackhal
RUN cd /go/src/github.com/CyrilPeponnet/slackhal && glide install
RUN cd /go/bin && go build github.com/CyrilPeponnet/slackhal

# final stage
FROM alpine
WORKDIR /slackhal
COPY --from=build-env /go/bin/slackhal /slackhal/

VOLUME /slackhal/

ENTRYPOINT ./slackhal
