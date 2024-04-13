FROM golang:1.9-stretch
MAINTAINER Lars Gierth <lgierth@ipfs.io>

# There is a copy of this Dockerfile called Dockerfile.fast,
# which is optimized for build time, instead of image size.
#
# Please keep these two Dockerfiles in sync.

ENV GX_IPFS ""
ENV SRC_DIR /go/src/github.com/ipfs/go-ipfs

COPY . $SRC_DIR

# Build the thing.
RUN cd $SRC_DIR \
  # Required for getting the HEAD commit hash via git rev-parse.
  && mkdir .git/objects \
  # Allows using a custom (i.e. local) IPFS API endpoint.
  && ([ -z "$GX_IPFS" ] || echo $GX_IPFS > /root/.ipfs/api) \
  # Build the thing.
  && make build

# Get the TLS CA certificates, they're not provided by busybox.
RUN apt-get install -y ca-certificates

# Now comes the actual target image, which aims to be as small as possible.
FROM busybox:1-glibc
MAINTAINER Lars Gierth <lgierth@ipfs.io>

# Get the ipfs binary, entrypoint script, and TLS CAs from the build container.
ENV SRC_DIR /go/src/github.com/ipfs/go-ipfs
COPY --from=0 $SRC_DIR/cmd/ipfs/ipfs /usr/local/bin/ipfs
COPY --from=0 $SRC_DIR/bin/container_daemon /usr/local/bin/start_ipfs
COPY --from=0 /etc/ssl/certs /etc/ssl/certs

# This shared lib (part of glibc) doesn't seem to be included with busybox.
COPY --from=0 /lib/x86_64-linux-gnu/libdl-2.24.so /lib/libdl.so.2

# This installs a very simple program acting as the init process.
# Makes sure signals are properly passed to the ipfs daemon process.
ENV TINI_VERSION v0.16.1
ADD https://github.com/krallin/tini/releases/download/$TINI_VERSION/tini /sbin/tini
RUN chmod +x /sbin/tini

# Ports for Swarm TCP, Swarm uTP, API, Gateway, Swarm Websockets
EXPOSE 4001
EXPOSE 4002/udp
EXPOSE 5001
EXPOSE 8080
EXPOSE 8081

# Create the fs-repo directory and switch to a non-privileged user.
ENV IPFS_PATH /data/ipfs
RUN mkdir -p $IPFS_PATH && adduser -D -h $IPFS_PATH -u 1000 -g 100 ipfs
USER ipfs

# Expose the fs-repo as a volume.
# start_ipfs initializes an fs-repo if none is mounted.
# Important this happens after the USER directive so permission are correct.
VOLUME $IPFS_PATH

# The default logging level
ENV IPFS_LOGGING ""

# This just makes sure that:
# 1. There's an fs-repo, and initializes one if there isn't.
# 2. The API and Gateway are accessible from outside the container.
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/start_ipfs"]

# Execute the daemon subcommand by default
CMD ["daemon", "--migrate=true"]
