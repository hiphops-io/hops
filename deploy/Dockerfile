# syntax=docker/dockerfile:1
# Build stage
FROM golang:alpine as builder

RUN apk add --no-cache --update git nodejs-current npm

WORKDIR /go/src/app

# Download dependencies separately to better take advantage of Docker cache
COPY go.mod .
COPY go.sum .
RUN go mod download

# Now we copy the code and do the actual build
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/app -v main.go

# Final stage
FROM alpine:3.18

RUN apk --no-cache add ca-certificates

# Creates a new user with uid 1000 and a home directory
RUN adduser -D -u 1000 hops
USER hops
WORKDIR /home/hops

COPY --chown=hops:hops --from=builder /go/bin/app ./hops

EXPOSE 8916

ENTRYPOINT [ "./hops" ]
CMD ["start"]
