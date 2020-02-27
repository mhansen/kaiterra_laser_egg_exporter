# Can compile on any architecture - thanks Go!

# Use the official Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.14 as builder

# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
COPY go.* ./
RUN go mod download

# Copy local code to the container image.
COPY *.go ./

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -mod=readonly -v -a kaiterra_laser_egg_exporter.go

FROM scratch
# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/kaiterra_laser_egg_exporter /
EXPOSE 9660

# Run the web service on container startup.
ENTRYPOINT ["/kaiterra_laser_egg_exporter"]
