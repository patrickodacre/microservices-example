# use the official golang image as the build environment.
FROM golang:1.21.3-alpine3.18 as builder

# set the working directory.
WORKDIR /usr/src/app

# copy the go module files and download dependencies.
COPY go.mod ./
# COPY go.mod go.sum ./
# run go mod download

# copy the source code.
COPY . .

# build the go app
RUN go build -o main .

# CMD ["/usr/src/app/main"]

# start from a fresh base image to create a lean production image.
FROM alpine:3.18

COPY --from=builder main /bin/main

CMD ["/bin/main"]
