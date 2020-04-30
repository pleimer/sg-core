#!/bin/sh

/bridge --stat-period 1 --verbose --gw_unix /smartgateway --amqp_url amqp://stf-default-interconnect.service-telemetry.svc.cluster.local:5672/collectd/telemetry &
/smart_gateway -promhost 0.0.0.0 unix -path /smartgateway
