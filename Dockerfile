FROM lscr.io/linuxserver/code-server:4.103.2

# System-wide packages
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y \
    curl wget ssh vim nano git ca-certificates build-essential \
    openssh-client gnupg unzip zip xz-utils \
    rsync coreutils procps \
    software-properties-common gcc gcc-9 \
    python-is-python3 python3.12-venv python3.12-dev \
    krb5-user libcurl4-openssl-dev

# Python env vars
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PYTHONIOENCODING=utf-8

# Python deps
RUN apt-get install -y \
    libssl-dev zlib1g-dev libbz2-dev libreadline-dev libsqlite3-dev \
    libffi-dev liblzma-dev tk-dev libdb-dev libncursesw5-dev \
    libgdbm-dev libc6-dev uuid-dev

# Install Python 3.11.12
WORKDIR /opt
RUN wget https://python.org/ftp/python/3.11.12/Python-3.11.12.tgz && \
    tar -xzf Python-3.11.12.tgz && \
    cd Python-3.11.12 && \
    ./configure &&  \
    make -j 4 && \
    make altinstall && \
    cd .. && \
    rm -rf Python-3.11.12.tgz

WORKDIR /

# Set Timezone
ENV TZ="Europe/Istanbul"
RUN ln -fs /usr/share/zoneinfo/Europe/Istanbul /etc/localtime && \
    dpkg-reconfigure -f noninteractive tzdata

# Extras
RUN apt-get install openjdk-8-jdk -y

