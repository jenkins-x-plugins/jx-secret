package wait_test

import (
	"testing"
	"time"

	"github.com/jenkins-x/jx-secret/pkg/cmd/vault/wait"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestVaultWait(t *testing.T) {
	var err error
	_, o := wait.NewCmdWait()

	o.PollDuration = 2 * time.Second
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
	}

	o.KubeClient = fake.NewSimpleClientset(kubeObjects...)

	err = o.Run()
	require.NoError(t, err, "failed to run")
}

func TestVaultWaitFails(t *testing.T) {
	var err error
	_, o := wait.NewCmdWait()

	o.PollDuration = 1 * time.Second

	o.KubeClient = fake.NewSimpleClientset()

	err = o.Run()
	require.Error(t, err, "expected failure")

	t.Logf("got expected failure waiting for vault: %s", err.Error())
}
