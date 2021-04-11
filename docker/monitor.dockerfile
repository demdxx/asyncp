FROM scratch

LABEL maintainer="Dmitry Ponomarev <demdxx@gmail.com>"

ENV APMON_REFRESH_INTERVAL=1s
ARG APMON_APPNAME
ARG APMON_STORAGE_CONNECT

ADD .build/apmonitor apmonitor

ENTRYPOINT [ "/apmonitor" ]