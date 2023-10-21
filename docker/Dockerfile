# Build stage
FROM golang:alpine as builder

RUN apk add --no-cache --update git nodejs-current npm

WORKDIR /go/src/app

# Download dependencies separately to better take advantage of Docker cache
COPY go.mod .
COPY go.sum .
RUN go mod download
RUN npm install -g pnpm

COPY console console
RUN cd console && pnpm install && pnpm run build

# Now we copy the code and do the actual build
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/app -v main.go

# Final stage
FROM alpine:3.18
RUN apk --no-cache add ca-certificates
COPY --from=builder /go/bin/app /hops

EXPOSE 8916

ENTRYPOINT [ "/hops" ]

CMD ["start"]
# TODO: Run as non-root?
