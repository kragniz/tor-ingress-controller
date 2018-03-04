/*
Copyright 2018 Louis Taylor <louis@kragniz.eu>.
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"

	"github.com/kragniz/tor-ingress-controller/tor"
)

const (
	annotationName           = "kubernetes.io/ingress.class"
	privateKeyAnnotationName = "kragniz.eu/tor-private-key"
)

type TorController struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller

	clientset *kubernetes.Clientset

	torCfg tor.TorConfiguration
	tor    tor.Tor
}

func NewTorController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller, clientset *kubernetes.Clientset) *TorController {
	return &TorController{
		informer:  informer,
		indexer:   indexer,
		queue:     queue,
		clientset: clientset,
		torCfg:    tor.NewTorConfiguration(),
	}
}

func (t *TorController) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := t.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two ingresses with the same key are never processed in
	// parallel.
	defer t.queue.Done(key)

	// Invoke the method containing the business logic
	err := t.syncTor(key.(string))
	// Handle the error if something went wrong during the execution of the business logic
	t.handleErr(err, key)
	return true
}

func (t *TorController) isTorIngress(ing *v1beta1.Ingress) bool {
	if class, exists := ing.Annotations[annotationName]; exists {
		return class == "tor"
	}
	return false
}

func (t *TorController) getTorPrivateKey(ing *v1beta1.Ingress) (*string, error) {
	if keyName, exists := ing.Annotations[privateKeyAnnotationName]; exists {
		secret, err := t.clientset.CoreV1().Secrets(ing.Namespace).Get(keyName, meta_v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		for _, item := range secret.StringData {
			return &item, nil
		}
	}
	return nil, nil
}

// syncTor updates the tor config with the current set of ingresses
func (t *TorController) syncTor(key string) error {
	obj, exists, err := t.indexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		fmt.Printf("Ingress %s does not exist anymore\n", key)
		t.torCfg.RemoveService(key)
		fmt.Println(t.torCfg.GetConfiguration())
		t.torCfg.SaveConfiguration()
		t.tor.Reload()
	} else {
		switch o := obj.(type) {
		case *v1beta1.Ingress:
			// Note that you also have to check the uid if you have a local controlled resource, which
			// is dependent on the actual instance, to detect that a Ingress was recreated with the same name
			fmt.Printf("Sync/Add/Update for Ingress %s, namespace %s\n", o.GetName(), o.GetNamespace())

			if !t.isTorIngress(o) {
				fmt.Println("this isn't a tor ingress")
				return nil
			}

			backend := o.Spec.Backend
			if backend == nil {
				fmt.Println("sorry, only basic backend supported")
			} else {
				service, err := t.clientset.CoreV1().Services(o.GetNamespace()).Get(backend.ServiceName, meta_v1.GetOptions{})
				if err != nil {
					fmt.Printf("service not found! %v", err)
				}

				clusterIP := service.Spec.ClusterIP

				s := t.torCfg.AddService(
					o.GetName(),
					backend.ServiceName,
					o.GetNamespace(),
					clusterIP,
					int(backend.ServicePort.IntVal),
					int(backend.ServicePort.IntVal),
				)

				secret, err := t.getTorPrivateKey(o)
				if err != nil {
					fmt.Printf("error fetching private key! %v", err)
				} else {
					if secret != nil {
						os.MkdirAll(s.ServiceDir, 0700)

						file, err := os.Create(path.Join(s.ServiceDir, "private-key"))
						if err != nil {
							return err
						}

						file.WriteString(*secret)
					}
				}

				fmt.Println(t.torCfg.GetConfiguration())
				t.torCfg.SaveConfiguration()
				t.tor.Reload()

				time.Sleep(5 * time.Second)

				fmt.Println("finding tor hostname")
				hostname, err := s.FindHostname()
				if err != nil {
					return err
				}

				fmt.Println("hostname found! ", hostname)

				ingressClient := t.clientset.ExtensionsV1beta1().Ingresses(o.Namespace)

				retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					o.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{{Hostname: hostname}}

					_, updateErr := ingressClient.UpdateStatus(o)
					if updateErr != nil {
						panic(fmt.Errorf("Update error: %v", updateErr))
					}
					return updateErr
				})
				if retryErr != nil {
					panic(fmt.Errorf("Update failed: %v", retryErr))
				}
			}
		}
	}
	return nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (t *TorController) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		t.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if t.queue.NumRequeues(key) < 5 {
		glog.Infof("Error syncing ingress %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		t.queue.AddRateLimited(key)
		return
	}

	t.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping ingress %q out of the queue: %v", key, err)
}

func (t *TorController) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer t.queue.ShutDown()
	glog.Info("Starting tor controller")

	go t.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, t.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(t.runWorker, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Stopping tor controller")
}

func (t *TorController) runWorker() {
	for t.processNextItem() {
	}
}

func main() {
	var kubeconfig string
	var master string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.Parse()

	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}

	// create the ingress watcher
	ingressListWatcher := cache.NewListWatchFromClient(clientset.ExtensionsV1beta1().RESTClient(), "ingresses", v1.NamespaceAll, fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the ingress key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Ingress than the version which was responsible for triggering the update.
	indexer, informer := cache.NewIndexerInformer(ingressListWatcher, &v1beta1.Ingress{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	controller := NewTorController(queue, indexer, informer, clientset)

	controller.tor.Start()

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	// Wait forever
	select {}
}
