package main

import (
	"database/sql"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	queryConnections = `select count(1) from pg_stat_activity;`
)

func (e *Exporter) collectDatabaseMetric(ch chan<- prometheus.Metric) bool {
	db, err := sql.Open("postgres", e.pg.connPostgresStr)
	if err != nil {
		level.Error(e.client.logger).Log("msg", "Error connect to db", "err", err)
	}
	defer db.Close()

	health := 1.0
	err = db.Ping()
	if err != nil {
		health = 0.0
	}

	var conns float64
	res := db.QueryRow(queryConnections)
	res.Scan(&conns)

	ch <- prometheus.MustNewConstMetric(
		databaseHealth, prometheus.GaugeValue, health,
	)
	ch <- prometheus.MustNewConstMetric(
		databaseConnections, prometheus.GaugeValue, conns,
	)

	return true
}
