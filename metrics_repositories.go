package main

import (
	"database/sql"

	"github.com/go-kit/kit/log/level"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	queryAllRepo = `
SELECT
    r.repository_id as repo_id,
    r.name as repo_name,
    r.pull_count as pull_count,
    count(al.log_id) as push_count
FROM
    repository as r
    JOIN access_log as al ON al.repo_name = r.name
WHERE
    al.operation = 'push'
GROUP By
    r.repository_id;`
	queryRepoTag = `SELECT
	count(a.id) as tag_count
FROM
	artifact as a
	JOIN repository as r ON a.repo = r.name
WHERE
	r.repository_id = $1;`
	queryImagePull = `
SELECT
	a.repo as repo_name,
	a.tag as tag_name,
	count(al.log_id) as pull_count
FROM
	access_log as al
	JOIN artifact as a ON a.repo = al.repo_name
	AND a.tag = repo_tag
WHERE
	al.operation = 'pull'
GROUP BY
	a.repo,
	a.tag;`
	queryProjectSize = `
SELECT
    p.name as project_name,
    sum(b.size) as size
FROM
    blob AS b
    JOIN artifact_blob AS ab ON ab.digest_blob = b.digest
    JOIN artifact AS a ON a.digest = digest_af
    JOIN repository AS r ON a.repo = r.name
    JOIN project AS p ON r.project_id = p.project_id
GROUP BY
    p.name;`
)

func (e *Exporter) collectRepositoriesMetric(ch chan<- prometheus.Metric, version string) bool {
	db, err := sql.Open("postgres", e.pg.connStr)
	if err != nil {
		level.Error(e.client.logger).Log("msg", "Error connect to db", "err", err)
	}
	defer db.Close()

	// 得到全部repo的信息
	type Repo struct {
		repo_id    string
		repo_name  string
		pull_count float64
		push_count float64
		tag_count  float64
	}

	res, err := db.Query(queryAllRepo)
	if err != nil {
		level.Error(e.client.logger).Log("msg", "Error get repo info", "err", err)
	}
	repo := &Repo{}
	for res.Next() {
		err := res.Scan(&repo.repo_id, &repo.repo_name, &repo.pull_count, &repo.push_count)
		if err != nil {
			level.Error(e.client.logger).Log("msg", "Error get repo info", "err", err)
		}

		res := db.QueryRow(queryRepoTag, repo.repo_id)
		res.Scan(&repo.tag_count)

		ch <- prometheus.MustNewConstMetric(
			repositoriesPullCount, prometheus.GaugeValue, repo.pull_count, repo.repo_name, repo.repo_id,
		)
		ch <- prometheus.MustNewConstMetric(
			repositoriesPushCount, prometheus.GaugeValue, repo.push_count, repo.repo_name, repo.repo_id,
		)
		ch <- prometheus.MustNewConstMetric(
			repositoriesTagsCount, prometheus.GaugeValue, repo.tag_count, repo.repo_name, repo.repo_id,
		)
	}

	// 得到全部image的信息
	type Image struct {
		repo_name  string
		tag_name   string
		pull_count float64
	}
	res, err = db.Query(queryImagePull)
	if err != nil {
		level.Error(e.client.logger).Log("msg", "Error get image data", "err", err)
	}
	image := &Image{}
	for res.Next() {
		res.Scan(&image.repo_name, &image.tag_name, &image.pull_count)
		ch <- prometheus.MustNewConstMetric(
			imagePullCount, prometheus.GaugeValue, image.pull_count, image.repo_name, image.tag_name,
		)
	}

	// 得到全部项目的占用空间
	type Project struct {
		project_name string
		size         float64
	}
	res, err = db.Query(queryProjectSize)
	if err != nil {
		level.Error(e.client.logger).Log("msg", "Error get project size", "err", err)
	}
	project := &Project{}
	var mb float64 = 1048576
	for res.Next() {
		res.Scan(&project.project_name, &project.size)
		ch <- prometheus.MustNewConstMetric(
			projectSize, prometheus.GaugeValue, project.size/mb, project.project_name,
		)
	}

	return true
}
