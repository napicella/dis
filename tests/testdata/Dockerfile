FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y sudo make wget curl zip unzip tar git tree gpg apt-utils gettext-base \
    && useradd -m -s /bin/bash dev \
    && echo "dev ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

USER dev
WORKDIR /home/dev