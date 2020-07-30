# These will be provided to the target
VERSION := `cat VERSION`
REVISION := `git rev-parse HEAD`
BRANCH := `git rev-parse --abbrev-ref HEAD`
USER := `whoami`
NOW := `date -u +'%F-%T-%Z'`
# Use linker flags to provide version/build settings to the target
LDFLAGS=-ldflags "-X=github.com/prometheus/common/version.Version=$(VERSION) \
-X=github.com/prometheus/common/version.Branch=$(BRANCH) \
-X=github.com/prometheus/common/version.Revision=$(REVISION) \
-X=github.com/prometheus/common/version.BuildDate=$(NOW) \
-X=github.com/prometheus/common/version.BuildUser=$(USER)"


build:
	@go build $(LDFLAGS) -o releases/harbor_exporter -mod vendor

dockerbuild:
	docker build --network host -t reg.ebcpaas.com/xiechuyu/harbor_exporter .

dockerpush:
	docker push reg.ebcpaas.com/xiechuyu/harbor_exporter
