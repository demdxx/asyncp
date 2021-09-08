# syntax=docker/dockerfile:1.2
FROM --platform=$TARGETPLATFORM scratch

LABEL maintainer="Dmitry Ponomarev <demdxx@gmail.com>"

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG APMON_APPNAME
ARG APMON_STORAGE_CONNECT
ENV APMON_REFRESH_INTERVAL=1s

ADD .build/${TARGETPLATFORM}/apmonitor /apmonitor

ENTRYPOINT [ "/apmonitor" ]