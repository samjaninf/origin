package controlplane

import (
	"context"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/upgrades"

	"github.com/blang/semver"
	apiconfigv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/origin/pkg/monitor"
	"github.com/openshift/origin/test/extended/util"
	"github.com/openshift/origin/test/extended/util/disruption"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewKubeAvailableWithNewConnectionsTest tests that the Kubernetes control plane remains available during and after a cluster upgrade.
func NewKubeAvailableWithNewConnectionsTest() upgrades.Test {
	restConfig, err := monitor.GetMonitorRESTConfig()
	utilruntime.Must(err)
	backendSampler, err := monitor.CreateKubeAPIMonitoringWithNewConnections(restConfig)
	utilruntime.Must(err)
	return disruption.NewBackendDisruptionTest(
		"[sig-api-machinery] Kubernetes APIs remain available for new connections",
		backendSampler,
	).WithAllowedDisruption(allowedAPIServerDisruption)
}

// NewOpenShiftAvailableNewConnectionsTest tests that the OpenShift APIs remains available during and after a cluster upgrade.
func NewOpenShiftAvailableNewConnectionsTest() upgrades.Test {
	restConfig, err := monitor.GetMonitorRESTConfig()
	utilruntime.Must(err)
	backendSampler, err := monitor.CreateOpenShiftAPIMonitoringWithNewConnections(restConfig)
	utilruntime.Must(err)
	return disruption.NewBackendDisruptionTest(
		"[sig-api-machinery] OpenShift APIs remain available for new connections",
		backendSampler,
	).WithAllowedDisruption(allowedAPIServerDisruption)
}

// NewOAuthAvailableNewConnectionsTest tests that the OAuth APIs remains available during and after a cluster upgrade.
func NewOAuthAvailableNewConnectionsTest() upgrades.Test {
	restConfig, err := monitor.GetMonitorRESTConfig()
	utilruntime.Must(err)
	backendSampler, err := monitor.CreateOAuthAPIMonitoringWithNewConnections(restConfig)
	utilruntime.Must(err)
	return disruption.NewBackendDisruptionTest(
		"[sig-api-machinery] OAuth APIs remain available for new connections",
		backendSampler,
	).WithAllowedDisruption(allowedAPIServerDisruption)
}

// NewKubeAvailableWithConnectionReuseTest tests that the Kubernetes control plane remains available during and after a cluster upgrade.
func NewKubeAvailableWithConnectionReuseTest() upgrades.Test {
	restConfig, err := monitor.GetMonitorRESTConfig()
	utilruntime.Must(err)
	backendSampler, err := monitor.CreateKubeAPIMonitoringWithConnectionReuse(restConfig)
	utilruntime.Must(err)
	return disruption.NewBackendDisruptionTest(
		"[sig-api-machinery] Kubernetes APIs remain available with reused connections",
		backendSampler,
	).WithAllowedDisruption(allowedAPIServerDisruption)
}

// NewOpenShiftAvailableWithConnectionReuseTest tests that the OpenShift APIs remains available during and after a cluster upgrade.
func NewOpenShiftAvailableWithConnectionReuseTest() upgrades.Test {
	restConfig, err := monitor.GetMonitorRESTConfig()
	utilruntime.Must(err)
	backendSampler, err := monitor.CreateOpenShiftAPIMonitoringWithConnectionReuse(restConfig)
	utilruntime.Must(err)
	return disruption.NewBackendDisruptionTest(
		"[sig-api-machinery] OpenShift APIs remain available with reused connections",
		backendSampler,
	).WithAllowedDisruption(allowedAPIServerDisruption)
}

// NewOAuthAvailableWithConnectionReuseTest tests that the OAuth APIs remains available during and after a cluster upgrade.
func NewOAuthAvailableWithConnectionReuseTest() upgrades.Test {
	restConfig, err := monitor.GetMonitorRESTConfig()
	utilruntime.Must(err)
	backendSampler, err := monitor.CreateOAuthAPIMonitoringWithConnectionReuse(restConfig)
	utilruntime.Must(err)
	return disruption.NewBackendDisruptionTest(
		"[sig-api-machinery] OAuth APIs remain available with reused connections",
		backendSampler,
	).WithAllowedDisruption(allowedAPIServerDisruption)
}

func allowedAPIServerDisruption(f *framework.Framework, totalDuration time.Duration) (*time.Duration, error) {
	// starting from 4.8, enforce the requirement that control plane remains available
	hasAllFixes, err := util.AllClusterVersionsAreGTE(semver.Version{Major: 4, Minor: 8}, f.ClientConfig())
	if err != nil {
		framework.Logf("Cannot require full control plane availability, some versions could not be checked: %v", err)
	}

	toleratedDisruption := 0.08
	switch controlPlaneTopology, _ := getTopologies(f); {
	case controlPlaneTopology == apiconfigv1.SingleReplicaTopologyMode:
		// we cannot avoid API downtime during upgrades on single-node control plane topologies (we observe around ~10% disruption)
		toleratedDisruption = 0.15

		// We observe even higher disruption on Azure single-node upgrades
		if framework.ProviderIs("azure") {
			toleratedDisruption = 0.23
		}
	case framework.ProviderIs("azure"), framework.ProviderIs("aws"), framework.ProviderIs("gce"):
		if hasAllFixes {
			framework.Logf("Cluster contains no versions older than 4.8, tolerating no disruption")
			toleratedDisruption = 0
		}
	}
	allowedDisruptionNanoseconds := int64(float64(totalDuration.Nanoseconds()) * toleratedDisruption)
	allowedDisruption := time.Duration(allowedDisruptionNanoseconds)

	return &allowedDisruption, nil
}

func getTopologies(f *framework.Framework) (controlPlaneTopology, infraTopology apiconfigv1.TopologyMode) {
	oc := util.NewCLIWithFramework(f)
	infra, err := oc.AdminConfigClient().ConfigV1().Infrastructures().Get(context.Background(),
		"cluster", metav1.GetOptions{})

	framework.ExpectNoError(err, "unable to determine cluster topology")

	return infra.Status.ControlPlaneTopology, infra.Status.InfrastructureTopology
}
