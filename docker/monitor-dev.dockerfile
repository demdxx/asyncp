FROM golang:latest

LABEL maintainer="Dmitry Ponomarev <demdxx@gmail.com>"

ARG APMON_APPNAME
ARG APMON_STORAGE_CONNECT
ENV APMON_REFRESH_INTERVAL=1s

ADD .build/apmonitor /apmonitor

ENTRYPOINT [ "/apmonitor" ]