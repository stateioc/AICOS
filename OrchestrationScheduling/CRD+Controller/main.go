/*
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
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"log"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	boxv1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	boxstatesetv1alpha1 "cncos.io/box-controller/pkg/apis/boxstatefulset/v1alpha1"
	clientset "cncos.io/box-controller/pkg/box-generated/clientset/versioned"
	informers "cncos.io/box-controller/pkg/box-generated/informers/externalversions"
	boxdeploymentclientset "cncos.io/box-controller/pkg/boxdeployment-generated/clientset/versioned"
	boxdeploymentinformers "cncos.io/box-controller/pkg/boxdeployment-generated/informers/externalversions"
	boxstatefulsetclientset "cncos.io/box-controller/pkg/boxstatefulset-generated/clientset/versioned"
	boxstatefulsetinformers "cncos.io/box-controller/pkg/boxstatefulset-generated/informers/externalversions"
	boxstatefulsetcontroller "cncos.io/box-controller/pkg/controller/box-statefulset"
	"cncos.io/box-controller/pkg/signals"
)

var (
	masterURL    string
	kubeconfig   string
	reSyncPeriod int
	worker       int
	kubeCfgQPS   int
	kubeCfgBurst int

	boxControllerEnable            bool
	boxDeploymentContrllerEnable   bool
	boxStatefulSetControllerEnable bool

	proxyScheme = runtime.NewScheme()
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the shutdown signal gracefully
	ctx := signals.SetupSignalHandler()
	logger := klog.FromContext(ctx)

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		logger.Error(err, "Error building kubeconfig")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	cfg.QPS = float32(kubeCfgQPS)
	cfg.Burst = kubeCfgBurst

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	boxClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	boxdeploymentClient, err := boxdeploymentclientset.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	boxStatefulSetClient, err := boxstatefulsetclientset.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*time.Duration(reSyncPeriod))
	boxInformerFactory := informers.NewSharedInformerFactory(boxClient, time.Second*time.Duration(reSyncPeriod))
	boxdeploymentInformerFactory := boxdeploymentinformers.NewSharedInformerFactory(boxdeploymentClient, time.Second*time.Duration(reSyncPeriod))
	boxStatefulSetInformerFactory := boxstatefulsetinformers.NewSharedInformerFactory(boxStatefulSetClient, time.Second*time.Duration(reSyncPeriod))

	if boxControllerEnable {
		controller := NewController(ctx, kubeClient, boxClient, boxdeploymentClient,
			kubeInformerFactory.Core().V1().Pods(),
			// kubeInformerFactory.Apps().V1().Deployments(),
			boxdeploymentInformerFactory.Cncos().V1alpha1().BoxDeployments(),
			boxInformerFactory.Cncos().V1alpha1().Boxes(),
		)

		go func() {
			if err = controller.Run(ctx, worker); err != nil {
				logger.Error(err, "Error running controller")
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}()
	}

	if boxDeploymentContrllerEnable {
		boxdpcontroller := NewBoxDeploymentController(ctx, kubeClient, boxClient, boxdeploymentClient,
			kubeInformerFactory.Core().V1().Pods(),
			// kubeInformerFactory.Apps().V1().Deployments(),
			boxdeploymentInformerFactory.Cncos().V1alpha1().BoxDeployments(),
			boxInformerFactory.Cncos().V1alpha1().Boxes(),
		)
		go func() {
			if err = boxdpcontroller.Run(ctx, worker); err != nil {
				logger.Error(err, "Error running controller")
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}()
	}

	if boxStatefulSetControllerEnable {
		boxStatefulSetController := boxstatefulsetcontroller.NewBoxStatefulSetController(ctx, kubeClient, boxClient, boxStatefulSetClient,
			boxStatefulSetInformerFactory.Cncos().V1alpha1().BoxStatefulSets(),
			kubeInformerFactory.Core().V1().PersistentVolumeClaims(),
			kubeInformerFactory.Apps().V1().ControllerRevisions(), boxInformerFactory.Cncos().V1alpha1().Boxes())
		go func() {
			if err = boxStatefulSetController.Run(ctx, worker); err != nil {
				logger.Error(err, "Error running BoxStatefulSet controller")
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}()

	}

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(ctx.done())
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(ctx.Done())
	boxInformerFactory.Start(ctx.Done())
	boxdeploymentInformerFactory.Start(ctx.Done())
	boxStatefulSetInformerFactory.Start(ctx.Done())

	<-ctx.Done()
	logger.Info("Shutting down workers")
}

func init() {
	log.SetFlags(log.Llongfile | log.Ldate | log.Lmicroseconds)
	log.SetPrefix("++++ ")

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.IntVar(&reSyncPeriod, "resync-period", 30, "resync period second.")
	flag.IntVar(&worker, "worker", 2, "worker num.")
	flag.BoolVar(&boxControllerEnable, "box-controller-enable", true, "box-controller-enable")
	flag.BoolVar(&boxDeploymentContrllerEnable, "boxdeployment-controller-enable", true, "boxdeployment-controller-enable")
	flag.BoolVar(&boxStatefulSetControllerEnable, "boxstatefulset-controller-enable", true, "boxstatefulset-controller-enable")
	flag.IntVar(&kubeCfgQPS, "kube-config-qps", 100, "Maximum QPS to the api-server from this client.")
	flag.IntVar(&kubeCfgBurst, "kube-config-burst", 200, "Maximum burst for throttle to the api-server from this client.")

	utilruntime.Must(boxv1alpha1.AddToScheme(proxyScheme))
	utilruntime.Must(boxstatesetv1alpha1.AddToScheme(proxyScheme))
	utilruntime.Must(scheme.AddToScheme(proxyScheme))
}
