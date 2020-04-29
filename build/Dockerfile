# --- build smart gateway ---
FROM centos:7 AS builder
ENV GOPATH=/go
ENV D=/go/src/github.com/infrawatch/sg

WORKDIR $D
COPY . $D/

RUN yum install epel-release -y && \
        yum update -y --setopt=tsflags=nodocs && \
        yum install git golang gcc make qpid-proton-c-devel --setopt=tsflags=nodocs -y && \
        yum clean all && \
        go build -o smart_gateway cmd/server/server.go && \
        cd generator && \
        make && \
        cd .. && \
        git clone https://github.com/infrawatch/sgbridge.git && \
        cd sgbridge && \
        make && \
        cd .. && \
        mv generator/gen /tmp && \
        mv smart_gateway /tmp/ && \
        mv sgbridge/bridge /tmp/

# --- end build, create smart gateway layer ---
FROM centos:7

LABEL io.k8s.display-name="Smart Gateway" \
      io.k8s.description="A component of the Service Telemetry Framework on the server side that ingests data from AMQP 1.x and provides a metrics scrape endpoint for Prometheus, and forwards events to ElasticSearch" \
      maintainer="Leif Madsen <leif@redhat.com>"

RUN yum install epel-release -y && \
        yum update -y --setopt=tsflags=nodocs && \
        yum install qpid-proton-c --setopt=tsflags=nodocs -y && \
        yum clean all && \
        rm -rf /var/cache/yum

COPY --from=builder /tmp/smart_gateway /
COPY --from=builder /tmp/gen /
COPY --from=builder /tmp/bridge /
COPY entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
