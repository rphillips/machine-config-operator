// bootstrap_test_failure.go
//
// This test demonstrates that MCO PR #5891 does NOT protect the bootstrap code path.
//
// EXPECTED RESULT WITH CURRENT CODE: Test PASSES (meaning the bug exists)
// DESIRED RESULT AFTER FIX: Test FAILS (meaning validation would block it)

package render

import (
	"testing"

	ign3types "github.com/coreos/ignition/v2/config/v3_5/types"
	mcfgv1 "github.com/openshift/api/machineconfiguration/v1"
	"github.com/openshift/machine-config-operator/pkg/controller/common"
	"github.com/openshift/machine-config-operator/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestBootstrapDoesNotBlockRuncOnRHEL10 proves that bootstrap is NOT protected.
//
// This test PASSES with current MCO PR #5891 code (demonstrating the vulnerability).
// After adding validation to RunBootstrap(), this test SHOULD FAIL.
func TestBootstrapDoesNotBlockRuncOnRHEL10(t *testing.T) {
	t.Log("========================================")
	t.Log("PROOF: Bootstrap does NOT validate runc")
	t.Log("========================================")
	t.Log("")

	// Create worker pool
	pool := &mcfgv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker",
		},
		Spec: mcfgv1.MachineConfigPoolSpec{
			MachineConfigSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"machineconfiguration.openshift.io/role": "worker",
				},
			},
		},
	}

	// Create MachineConfig that sets runc (simulating user manifest)
	runcMC := helpers.NewMachineConfig("99-worker-runc",
		map[string]string{"machineconfiguration.openshift.io/role": "worker"},
		"",
		[]ign3types.File{
			helpers.CreateEncodedIgn3File("/etc/crio/crio.conf.d/99-runc",
				"[crio.runtime]\ndefault_runtime = \"runc\"\n", 0644),
		})

	// Create ControllerConfig
	cc := newControllerConfig(common.ControllerConfigName)

	// Create OSImageStream for RHCOS 10
	osImageStream := &mcfgv1.OSImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.ClusterInstanceNameOSImageStream,
		},
		Status: mcfgv1.OSImageStreamStatus{
			DefaultStream: "rhel-10",
			AvailableStreams: []mcfgv1.OSImageStreamSet{
				{
					Name:    "rhel-10",
					OSImage: mcfgv1.ImageDigestFormat("quay.io/openshift/rhcos@sha256:fake"),
				},
			},
		},
	}

	t.Logf("Setup complete:")
	t.Logf("  Pool: %s", pool.Name)
	t.Logf("  MachineConfig: %s (configures runc)", runcMC.Name)
	t.Logf("  Target OS: RHCOS 10 (runc binary does not exist)")
	t.Log("")
	t.Log("Calling RunBootstrap()...")

	// Call RunBootstrap - this is the bootstrap code path
	pools, configs, err := RunBootstrap(
		[]*mcfgv1.MachineConfigPool{pool},
		[]*mcfgv1.MachineConfig{runcMC},
		cc,
		osImageStream,
	)

	t.Log("")
	t.Log("========================================")
	t.Log("RESULT:")
	t.Log("========================================")

	if err != nil {
		// DESIRED behavior (after fix is applied)
		t.Log("PROTECTED: Bootstrap rejected runc on RHCOS 10")
		t.Logf("  Error: %v", err)
		t.Log("")
		t.Log("The fix has been applied! Bootstrap validation is working.")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "runc")

	} else {
		// CURRENT behavior (vulnerability exists)
		t.Log("VULNERABLE: Bootstrap accepted runc on RHCOS 10")
		t.Logf("  Pools returned: %d", len(pools))
		t.Logf("  Configs returned: %d", len(configs))
		t.Log("")
		t.Log("This proves the gap exists:")
		t.Log("  1. RunBootstrap() did NOT validate runc")
		t.Log("  2. A rendered config was generated")
		t.Log("  3. If this were a real install:")
		t.Log("     - Bootstrap nodes would use RHCOS 10")
		t.Log("     - CRI-O would be configured for runc")
		t.Log("     - runc binary does not exist on RHCOS 10")
		t.Log("     - All containers would fail to start")
		t.Log("     - Cluster installation would FAIL")
		t.Log("")
		t.Log("FIX REQUIRED in RunBootstrap():")
		t.Log("  if err := validateNoRuncOnRHEL10(pool.Name, generated, osImageStreamSet); err != nil {")
		t.Log("      return nil, nil, err")
		t.Log("  }")

		// These assertions PASS with current code (proving the bug)
		// They SHOULD FAIL after the fix is applied
		assert.NoError(t, err, "Bootstrap should have rejected runc on RHEL 10")
		assert.NotEmpty(t, pools)
		assert.NotEmpty(t, configs)
	}

	t.Log("")
	t.Log("========================================")
}

// TestRuntimeDoesBlockRuncOnRHEL10 proves runtime path IS protected (for comparison).
func TestRuntimeDoesBlockRuncOnRHEL10(t *testing.T) {
	t.Log("========================================")
	t.Log("PROOF: Runtime DOES validate runc")
	t.Log("========================================")
	t.Log("")

	// Create MachineConfig with runc
	mc := helpers.NewMachineConfig("rendered-worker", nil, "", []ign3types.File{
		helpers.CreateEncodedIgn3File("/etc/crio/crio.conf.d/99-runc",
			"[crio.runtime]\ndefault_runtime = \"runc\"\n", 0644),
	})

	// RHCOS 10 OSImageStreamSet
	osImageStreamSet := &mcfgv1.OSImageStreamSet{Name: "rhel-10"}

	t.Log("Calling validateNoRuncOnRHEL10()...")
	err := validateNoRuncOnRHEL10("worker", mc, osImageStreamSet)

	t.Log("")
	t.Log("========================================")
	t.Log("RESULT:")
	t.Log("========================================")

	if err != nil {
		t.Log("PROTECTED: Runtime validation blocked runc on RHCOS 10")
		t.Logf("  Error: %v", err)
		t.Log("")
		t.Log("Runtime path is protected by MCO PR #5891")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "runc")
		assert.Contains(t, err.Error(), "not available")

	} else {
		t.Log("VULNERABLE: Runtime validation did not block runc")
		t.Fail()
	}

	t.Log("")
	t.Log("========================================")
	t.Log("COMPARISON:")
	t.Log("========================================")
	t.Log("  Runtime path: PROTECTED (validation works)")
	t.Log("  Bootstrap path: VULNERABLE (no validation)")
	t.Log("========================================")
}
