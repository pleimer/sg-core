#!/bin/sh

/bridge --stat_period 1 --verbose --gw_unix /var/tmp/smartgateway --amqp_url amqp://stf-default-interconnect.service-telemetry.svc.cluster.local:5672/collectd/telemetry &
/smart_gateway -promhost 0.0.0.0 unix -path /var/tmp/smartgateway
