package box_statefulset

import (
	"context"
	"flag"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	boxv1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	boxstatefulsetv1alpha1 "cncos.io/box-controller/pkg/apis/boxstatefulset/v1alpha1"
	clientset "cncos.io/box-controller/pkg/box-generated/clientset/versioned"
	informers "cncos.io/box-controller/pkg/box-generated/informers/externalversions"
	boxstatefulsetclientset "cncos.io/box-controller/pkg/boxstatefulset-generated/clientset/versioned"
	boxstatefulsetinformers "cncos.io/box-controller/pkg/boxstatefulset-generated/informers/externalversions"
)

var cfg *rest.Config
var testEnv *envtest.Environment
var testScheme = runtime.NewScheme()
var controllerDone context.CancelFunc
var controller *BoxStatefulSetController
var k8sClient client.Client

func TestBoxStatefulSet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BoxStatefulSet Controller Suite")
}

const (
	kubeConfigPath = "/Users/chenkun/Desktop/k8s/config-102-163"
)

var _ = BeforeSuite(func(done Done) {

	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))
	rand.Seed(time.Now().UnixNano())
	By("bootstrapping test environment")

	k8sconfig := flag.String("k8sconfig", kubeConfigPath, "kubernetes test")
	config, _ := clientcmd.BuildConfigFromFlags("", *k8sconfig)

	yamlPath := filepath.Join("../../../../..", "crd")
	testEnv = &envtest.Environment{
		ControlPlaneStartTimeout: time.Minute,
		ControlPlaneStopTimeout:  time.Minute,
		UseExistingCluster:       pointer.BoolPtr(true),
		CRDDirectoryPaths:        []string{yamlPath},
		Config:                   config,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = boxv1alpha1.SchemeBuilder.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())

	err = boxstatefulsetv1alpha1.SchemeBuilder.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())

	err = scheme.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())

	//k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	//Expect(err).ToNot(HaveOccurred())
	//Expect(k8sClient).ToNot(BeNil())

	var ctx context.Context
	ctx, controllerDone = context.WithCancel(context.Background())

	kubeClient, err := kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	boxClient, err := clientset.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	boxStatefulSetClient, err := boxstatefulsetclientset.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*time.Duration(30))
	boxInformerFactory := informers.NewSharedInformerFactory(boxClient, time.Second*time.Duration(30))
	boxStatefulSetInformerFactory := boxstatefulsetinformers.NewSharedInformerFactory(boxStatefulSetClient, time.Second*time.Duration(30))

	go func() {
		controller = NewBoxStatefulSetController(ctx, kubeClient, boxClient, boxStatefulSetClient,
			boxStatefulSetInformerFactory.Cncos().V1alpha1().BoxStatefulSets(),
			kubeInformerFactory.Core().V1().PersistentVolumeClaims(),
			kubeInformerFactory.Apps().V1().ControllerRevisions(),
			boxInformerFactory.Cncos().V1alpha1().Boxes())
		Expect(controller).ToNot(BeNil())
		//err = controller.Run(ctx, 2)
		//Expect(err).NotTo(HaveOccurred())
	}()

	kubeInformerFactory.Start(ctx.Done())
	boxInformerFactory.Start(ctx.Done())
	boxStatefulSetInformerFactory.Start(ctx.Done())

	// start the controller in the background so that new componentRevisions are created

	close(done)

}, 120)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	controllerDone()
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
