# See https://github.com/openshift/source-to-image/tree/master/examples
# for information about custon Source to Image (S2I) builder images.

FROM fedora:24

MAINTAINER Konrad Kleine <kkleine@redhat.com>

ENV LANG=en_US.utf8

RUN dnf install -y \
      findutils \
      git \
      golang \
      make \
      mercurial \
      procps-ng \
      tar \
      wget \
      which \
    && dnf clean all

# Get glide for Go package management
RUN cd /tmp \
    && wget https://github.com/Masterminds/glide/releases/download/v0.11.0/glide-v0.11.0-linux-amd64.tar.gz \
    && tar xvzf glide-v*.tar.gz \
    && mv linux-amd64/glide /usr/bin \
    && rm -rfv glide-v* linux-amd64

# Export the environment variable that provides information about the application.
# Replace this with the according version for your application.
ENV ALMIGHTY_CORE_VERSION=0.0.1

# Set the labels that are used for OpenShift to describe the builder image.
LABEL io.k8s.description="almighty core source to image (S2I) builder image" \
    io.k8s.display-name="almighty core 0.0.1" \
    io.openshift.expose-services="8080:http" \
    io.openshift.tags="builder,almighty,go" \
    # this label tells s2i where to find its mandatory scripts
    # (run, assemble, save-artifacts)
    io.openshift.s2i.scripts-url="image:///usr/libexec/s2i"

# Copy the S2I scripts to /usr/libexec/s2i since we set the label that way
COPY  ["run", "assemble", "save-artifacts", "usage", "/usr/libexec/s2i/"]

# Modify the usage script in your application dir to inform the user how to run
# this image.
CMD ["/usr/libexec/s2i/usage"]
