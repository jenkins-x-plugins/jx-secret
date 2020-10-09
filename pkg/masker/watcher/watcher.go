package watcher

import (
	"fmt"
	"sync"
	"time"

	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-kube-client/v3/pkg/kubeclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/masker"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

type Options struct {
	Namespaces []string
	KubeClient kubernetes.Interface

	replaceWordMap     map[string]map[string]string
	replaceWordMapLock sync.Mutex
	loggedMessages     map[string]bool
}

// Validate verifies things are setup correctly
func (o *Options) Validate() error {
	if o.replaceWordMap == nil {
		o.replaceWordMap = map[string]map[string]string{}
	}
	if o.loggedMessages == nil {
		o.loggedMessages = map[string]bool{}
	}
	var err error
	o.KubeClient, err = kube.LazyCreateKubeClient(o.KubeClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube client")
	}

	currentNS, err := kubeclient.CurrentNamespace()
	if err != nil {
		return errors.Wrapf(err, "failed to find current namespace")
	}

	if stringhelpers.StringArrayIndex(o.Namespaces, currentNS) < 0 {
		o.Namespaces = append(o.Namespaces, currentNS)
	}
	return nil
}

// Run runs the watching masker
func (o *Options) Run() error {
	stop := make(chan struct{})
	err := o.RunWithChannel(stop)
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	// Wait forever
	select {}
}

// RunWithChannel runs with the given channel
func (o *Options) RunWithChannel(stop chan struct{}) error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	log.Logger().Info("starting secret watching masker")

	for _, ns := range o.Namespaces {
		secret := &corev1.Secret{}
		log.Logger().Infof("Watching for Secret resources in namespace %s", ns)
		listWatch := cache.NewListWatchFromClient(o.KubeClient.CoreV1().RESTClient(), "secrets", ns, fields.Everything())
		kube.SortListWatchByName(listWatch)
		_, ctrl := cache.NewInformer(
			listWatch,
			secret,
			time.Minute*10,
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					o.onSecret(ns, obj)
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					o.onSecret(ns, newObj)
				},
				DeleteFunc: func(obj interface{}) {
				},
			},
		)
		go ctrl.Run(stop)
	}
	return nil
}

func (o *Options) onSecret(ns string, obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		log.Logger().Infof("Object is not an Secret %#v", obj)
		return
	}
	o.UpsertSecret(ns, secret)
}

// UpsertSecret upserts the secret in the replace words
func (o *Options) UpsertSecret(ns string, secret *corev1.Secret) {
	if secret != nil {
		client := &masker.Client{
			LogFn: func(text string) {
				if o.loggedMessages[text] == false {
					o.loggedMessages[text] = true
					log.Logger().Info(text)
				}
			},
		}
		err := client.LoadSecret(secret)
		if err != nil {
			log.Logger().Warnf("failed to load Secret %s namespace %s: %s", secret.Name, ns, err.Error())
			return
		}
		fullName := fmt.Sprintf("%s/%s", ns, secret.Name)

		o.replaceWordMapLock.Lock()
		o.replaceWordMap[fullName] = client.ReplaceWords
		o.replaceWordMapLock.Unlock()
	}
}

// GetClient returns the masker client for all the current secrets
func (o *Options) GetClient() *masker.Client {
	allWords := map[string]string{}

	o.replaceWordMapLock.Lock()
	for _, words := range o.replaceWordMap {
		for k, w := range words {
			allWords[k] = w
		}
	}
	o.replaceWordMapLock.Unlock()
	return &masker.Client{ReplaceWords: allWords}
}
