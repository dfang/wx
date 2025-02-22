# Accept the Go version for the image to be set as a build argument.
# Default to Go 1.12
# ARG GO_VERSION=1.12
# Please provide a source image with `from` prior to commit
# before Docker version 17.05, FROM instruction must be on the first line

# First stage: build the executable.
# FROM golang:${GO_VERSION}-alpine AS build_base
FROM golang:1.12-alpine AS builder


# Create the user and group files that will be used in the running container to
# run the process as an unprivileged user.
RUN mkdir /user && \
  echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
  echo 'nobody:x:65534:' > /user/group

# Install the Certificate-Authority certificates for the app to be able to make
# calls to HTTPS endpoints.
# Git is required for fetching the dependencies.
RUN apk add --no-cache ca-certificates git

# Set the working directory outside $GOPATH to enable the support for modules.
WORKDIR /src

# Fetch dependencies first; they are less susceptible to change on every build
# and will therefore be cached for speeding up the next build
COPY ./app/src/go.mod ./app/src/go.sum ./
RUN go mod download

# Import the code from the context.
COPY ./app/src/ ./

# Build the executable to `/app`. Mark the build as statically linked.
RUN CGO_ENABLED=0 go build \
  -installsuffix 'static' \
  -o /app .

# Final stage: the running container.
FROM scratch AS final

# Import the user and group files from the first stage.
COPY --from=builder /user/group /user/passwd /etc/

# Import the Certificate-Authority certificates for enabling HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import the compiled executable from the first stage.
COPY --from=builder /app /src/app
# COPY --from=builder /src/static /src/static
# COPY ./app/src/static /src/

# Declare the port on which the webserver will be exposed.
# As we're going to run the executable as an unprivileged user, we can't bind
# to ports below 1024.
EXPOSE 8080

# Perform any further action as an unprivileged user.
USER nobody:nobody

# Run the compiled binary.
ENTRYPOINT ["/src/app"]
