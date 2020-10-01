package shell_test

import (
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-secret/pkg/cmd/vault/shell"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestVaultShell(t *testing.T) {
	var err error
	_, o := shell.NewCmdVaultShell()

	ns := o.Namespace

	kubeObjects := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      o.PodName,
				Namespace: ns,
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
				Namespace: ns,
			},
			Data: map[string][]byte{
				"vault-root": []byte("dummyValue"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vault-tls",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"ca.crt": []byte("dummyValue"),
			},
		},
	}

	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(kubeObjects...)

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	err = o.Run()
	require.NoError(t, err, "failed to run edit")

	for _, c := range runner.OrderedCommands {
		t.Logf("ran %s\n", c.CLI())
	}
}
