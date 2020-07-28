package main

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

func (e *Exporter) collectSystemVolumesMetric(ch chan<- prometheus.Metric) bool {
	type systemVolumesMetric struct {
		Storage struct {
			Total float64
			Free  float64
		}
	}
	var data systemVolumesMetric
	pods, err := e.kubeClient.client.CoreV1().Pods(e.kubeClient.namespace).List(metav1.ListOptions{LabelSelector: "component=registry"})
	if err != nil {
		level.Error(e.client.logger).Log("msg", "Error getting registry pod", "err", err)
	}
	var targetPod v1.Pod
	for _, pod := range pods.Items {
		if !strings.Contains(pod.Name, "registry") {
			continue
		}
		targetPod = pod
	}

	req := e.kubeClient.client.CoreV1().RESTClient().Post().Resource("pods").
		Namespace(e.kubeClient.namespace).
		Name(targetPod.Name).
		SubResource("exec").
		VersionedParams(
			&v1.PodExecOptions{
				Container: "registryctl",
				Command: []string{
					"sh",
					"-c",
					"df " + e.opts.storage,
				},
				Stdin:  false,
				Stdout: true,
				Stderr: false,
				TTY:    false,
			}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(e.kubeClient.config, "POST", req.URL())
	if err != nil {
		level.Error(e.client.logger).Log("msg", "Error building remote exec request", "err", err)
	}
	var stdout bytes.Buffer
	exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: nil,
		Tty:    false,
	})

	result := strings.Split(strings.TrimSpace(strings.Split(stdout.String(), "\n")[1]), " ")
	var values []string
	for _, val := range result {
		if val != "" {
			values = append(values, val)
		}
	}
	var mb float64 = 1048576
	data.Storage.Free, err = strconv.ParseFloat(values[3], 64)
	if err != nil {
		level.Error(e.client.logger).Log("msg", "Error format free", "err", err)
	}
	data.Storage.Free = data.Storage.Free / mb

	data.Storage.Total, err = strconv.ParseFloat(values[1], 64)
	if err != nil {
		level.Error(e.client.logger).Log("msg", "Error format total", "err", err)
	}
	data.Storage.Total = data.Storage.Total / mb

	ch <- prometheus.MustNewConstMetric(
		systemVolumes, prometheus.GaugeValue, data.Storage.Total, "total",
	)
	ch <- prometheus.MustNewConstMetric(
		systemVolumes, prometheus.GaugeValue, data.Storage.Free, "free",
	)

	return true
}
