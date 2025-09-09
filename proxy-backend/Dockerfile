FROM golang:1.24.4 AS build-stage

ENV GOOS=linux
ENV CGO_ENABLED=0

ARG APP_NAME
ARG CMD_PATH

ENV APP_NAME=${APP_NAME}
ENV CMD_PATH=${CMD_PATH}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# COPY *.go ./
COPY . .

RUN go build -o ${APP_NAME} ${CMD_PATH}/main.go

FROM alpine:latest
ARG APP_NAME
ENV APP_NAME=${APP_NAME}
ARG GIT_SHA
ENV GIT_SHA=${GIT_SHA}

WORKDIR /app
COPY --from=build-stage /app/${APP_NAME} .

RUN apk add --no-cache tzdata
RUN cp /usr/share/zoneinfo/Europe/Istanbul /etc/localtime && echo 'Europe/Istanbul' > /etc/timezone
RUN TZ=Europe/Istanbul
