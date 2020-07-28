module github.com/c4po/harbor_exporter

go 1.13

require (
	github.com/go-kit/kit v0.9.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.4.0
	github.com/prometheus/common v0.9.1
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
)

replace (
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.0
)
