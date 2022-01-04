FROM ubuntu:16.04 as prod

WORKDIR /opt/app

COPY gateway /opt/app/
# COPY config /opt/app/

RUN chmod 755 /opt/app/gateway


CMD ["/bin/sh", "-c", "/opt/app/gateway"]