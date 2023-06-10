---
title: "Metrics"
---

# `kwokctl` Metrics

{{< hint "info" >}}

This document walks you through how to enable Metric on a `kwokctl` cluster

{{< /hint >}}

## Create a cluster with Prometheus

``` bash
kwokctl create cluster --prometheus-port 9090
```

## Create Grafana dashboard with Prometheus data source

``` bash
docker run -d --name=grafana -p 3000:3000 docker.io/grafana/grafana:9.4.7
```

1. Open your web browser and go to [http://localhost:3000]
2. On the login page, enter `admin` for username and password
3. Add the Prometheus data source, `http://host.docker.internal:9090`, on Grafana
4. Import via [grafana.com code] `16248` on Grafana

Now you can see the Grafana dashboard for the cluster.

[grafana.com code]: https://grafana.com/grafana/dashboards/16248
[http://localhost:3000]: http://localhost:3000
