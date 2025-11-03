package main

import (
	"context"
	"dagger/dex/internal/dagger"
	"fmt"
)

func (m *Dex) Test() *Test {
	return &Test{
		Source: m.Source,
	}
}

type Test struct {
	// +private
	Source *dagger.Directory
}

func (m *Test) chart() *dagger.HelmChart {
	chart := dag.Git("https://github.com/dexidp/helm-charts.git").Branch("master").Tree()
	chart = chart.Directory("charts/dex")
	// chart := m.Source.Directory("deploy/charts/dex")

	return dag.Helm().Chart(chart)
}

func (m *Test) Lint(ctx context.Context) (string, error) {
	return m.chart().Lint().Stdout(ctx)
}

var k3sVersions = map[string]string{
	"latest": "latest",
	"1.30":   "v1.30.13-k3s1",
	"1.31":   "v1.31.12-k3s1",
	"1.32":   "v1.32.8-k3s1",
	"1.33":   "v1.33.4-k3s1",
	"1.34":   "v1.34.1-rc1-k3s1",
}

func (m *Test) HelmChart(
	ctx context.Context,

	// +default="latest"
	kubeVersion string,
) error {
	k3sVersion, ok := k3sVersions[kubeVersion]
	if !ok {
		return fmt.Errorf("unsupported kube version: %s", kubeVersion)
	}

	k8s := dag.K3S("test", dagger.K3SOpts{
		Image: fmt.Sprintf("rancher/k3s:%s", k3sVersion),
	})

	_, err := k8s.Server().Start(ctx)
	if err != nil {
		return err
	}

	result, err := m.chart().Package().
		WithKubeconfigFile(k8s.Config()).
		Install("demo", dagger.HelmPackageInstallOpts{
			Wait:    true,
			Timeout: "1m",
			Values: []*dagger.File{
				m.chart().Directory().File("ci/test-values.yaml"),
			},
		}).
		Test(ctx, dagger.HelmReleaseTestOpts{
			Logs: true,
		})
	fmt.Println(result)
	if err != nil {
		return err
	}

	return nil
}
