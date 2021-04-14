package wait_test

import (
	"testing"
	"time"

	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/vault/wait"
	"github.com/jenkins-x-plugins/jx-secret/pkg/vaults"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

const testNs = "jx"

func TestVaultWait(t *testing.T) {
	var err error
	_, o := wait.NewCmdWait()

	o.WaitDuration = 2 * time.Second

	kubeObjects := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      o.PodName,
				Namespace: testNs,
				Labels: map[string]string{
					"app": "cheese",
				},
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vault-unseal-keys",
				Namespace: vaults.DefaultVaultNamespace,
			},
			Data: map[string][]byte{
				"vault-root": []byte("dummyValue"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vault-tls",
				Namespace: vaults.DefaultVaultNamespace,
			},
			Data: map[string][]byte{
				"ca.crt": []byte("dummyValue"),
			},
		},
	}

	o.Namespace = testNs
	o.KubeClient = fake.NewSimpleClientset(kubeObjects...)

	runner := &fakerunner.FakeRunner{
		CommandRunner: func(cmd *cmdrunner.Command) (string, error) {
			args := cmd.Args
			if len(args) > 2 && args[0] == "kv" && args[1] == "list" {
				return "ok", nil
			}
			return "", nil
		},
	}
	o.CommandRunner = runner.Run
	o.QuietCommandRunner = runner.Run

	err = o.Run()
	require.NoError(t, err, "failed to run")
}

func TestVaultWaitFails(t *testing.T) {
	var err error
	_, o := wait.NewCmdWait()

	o.WaitDuration = 1 * time.Second

	o.Namespace = testNs
	o.KubeClient = fake.NewSimpleClientset()

	err = o.Run()
	require.Error(t, err, "expected failure")

	t.Logf("got expected failure waiting for vault: %s", err.Error())
}
