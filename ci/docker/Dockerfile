FROM ubuntu:16.04
MAINTAINER Evan Brown <evanbrown@google.com>

# Update base image
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8
RUN locale-gen en_US.UTF-8; dpkg-reconfigure locales
RUN apt-get update; apt-get -y upgrade; apt-get clean

# Install dependencies
RUN apt-get -y install build-essential git-core curl wget jq sudo; apt-get clean

# Install Golang
ENV GOLANG_VERSION 1.7.1
RUN curl -sSL https://storage.googleapis.com/golang/go${GOLANG_VERSION}.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV GOROOT /usr/local/go
ENV PATH $PATH:$GOROOT/bin

# Instal chruby
RUN mkdir /tmp/chruby && \
    cd /tmp && \
    curl https://codeload.github.com/postmodern/chruby/tar.gz/v0.3.9 | tar -xz && \
    cd /tmp/chruby-0.3.9 && \
    sudo ./scripts/setup.sh && \
    rm -rf /tmp/chruby

# Install ruby-install
RUN mkdir /tmp/ruby-install && \
    cd /tmp && \
    curl https://codeload.github.com/postmodern/ruby-install/tar.gz/v0.5.0 | tar -xz && \
    cd /tmp/ruby-install-0.5.0 && \
    sudo make install && \
    rm -rf /tmp/ruby-install


# Set default ruby
RUN ruby-install ruby 2.1.2 && \
    cp /etc/profile.d/chruby.sh /etc/profile.d/chruby-with-ruby-2.1.2.sh && \
    echo "chruby 2.1.2" >> /etc/profile.d/chruby-with-ruby-2.1.2.sh

# Install Bundler and BOSH CLI
RUN /bin/bash -l -c "gem install bundler bosh_cli --no-ri --no-rdoc"