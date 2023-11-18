# use the official golang image as the build environment.
FROM golang:1.21.3-alpine3.18 as builder

RUN apk update && apk add gcc \
                          libc-dev \
                          libffi-dev
# set the working directory.
WORKDIR /usr/src/app

# copy the go module files and download dependencies.
COPY go.mod go.sum ./
# COPY go.mod go.sum ./
# run go mod download


# Kafka Go client is based on the C library librdkafka
ENV CGO_ENABLED 1
# ENV GOFLAGS -mod=vendor
ENV GOOS=linux
ENV GOARCH=amd64

# RUN export GO111MODULE=on

# copy the source code.
COPY ./cmd/. .

# build the go app
RUN go build -tags musl -o /bin/main .

# start from a fresh base image to create a lean production image.
FROM alpine:3.18
COPY --from=builder /bin/main /bin/main
CMD ["/bin/main"]
