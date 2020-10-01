package wait_test

import (
	"context"
	"testing"
	"time"

	"github.com/jenkins-x/jx-secret/pkg/cmd/wait"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestWait(t *testing.T) {
	var err error
	_, o := wait.NewCmdWait()
	scheme := runtime.NewScheme()

	ns := "jx"

	kubeObjects := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "knative-docker-user-pass",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"username": []byte("dummyValue"),
				"password": []byte("dummyPassword"),
			},
		},
	}
	dynObjects := testsecrets.LoadExtSecretDir(t, ns, "test_data")

	fakeDynClient := dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(kubeObjects...)

	valid, err := o.WaitCheck()
	require.NoError(t, err, "failed to run verify")
	require.False(t, valid, "should not be valid yet")

	// now lets make things valid
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lighthouse-oauth-token",
			Namespace: ns,
		},
		Data: map[string][]byte{
			"oauth": []byte("dummyValue"),
		},
	}
	_, err = o.KubeClient.CoreV1().Secrets(ns).Create(context.TODO(), secret, metav1.CreateOptions{})
	require.NoError(t, err, "failed to create Secret %#v")

	valid, err = o.WaitCheck()
	require.NoError(t, err, "failed to run verify")
	require.True(t, valid, "should be valid now")

	o.Timeout = time.Millisecond
	o.PollPeriod = time.Nanosecond
	err = o.Run()
	require.NoError(t, err, "run should not return an error")
}
