FROM golang:1.24.4
ARG UID
ENV UID=${UID}
ARG GID
ENV GID=${GID}
ARG USERNAME
ENV USERNAME=${USERNAME}

RUN DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata
RUN cp /usr/share/zoneinfo/Europe/Istanbul /etc/localtime && echo 'Europe/Istanbul' > /etc/timezone
RUN TZ=Europe/Istanbul

ENV GOOS=linux
ENV CGO_ENABLED=0

WORKDIR /app