package box_statefulset

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//func TestName(t *testing.T) {
//	k8sconfig := flag.String("k8sconfig", kubeConfigPath, "kubernetes test")
//	config, _ := clientcmd.BuildConfigFromFlags("", *k8sconfig)
//
//	kubeClient, err := kubernetes.NewForConfig(config)
//	Expect(err).NotTo(HaveOccurred())
//
//	boxClient, err := clientset.NewForConfig(config)
//	Expect(err).NotTo(HaveOccurred())
//
//	boxStatefulSetClient, err := boxstatefulsetclientset.NewForConfig(config)
//	Expect(err).NotTo(HaveOccurred())
//
//	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*time.Duration(30))
//	boxInformerFactory := informers.NewSharedInformerFactory(boxClient, time.Second*time.Duration(30))
//	boxStatefulSetInformerFactory := boxstatefulsetinformers.NewSharedInformerFactory(boxStatefulSetClient, time.Second*time.Duration(30))
//
//	ctx := signals.SetupSignalHandler()
//
//	go func() {
//		controller = NewBoxStatefulSetController(ctx, kubeClient, boxClient, boxStatefulSetClient,
//			boxStatefulSetInformerFactory.Cncos().V1alpha1().BoxStatefulSets(),
//			kubeInformerFactory.Core().V1().PersistentVolumeClaims(),
//			kubeInformerFactory.Apps().V1().ControllerRevisions(),
//			boxInformerFactory.Cncos().V1alpha1().Boxes())
//		Expect(controller).ToNot(BeNil())
//		err = controller.Run(ctx, 2)
//		Expect(err).NotTo(HaveOccurred())
//	}()
//
//	go func() {
//		time.Sleep(5 * time.Second)
//		key := "chenkun/boxsts-test"
//		err := controller.sync(ctx, key)
//		Expect(err).Should(BeNil())
//	}()
//
//	kubeInformerFactory.Start(ctx.Done())
//	boxInformerFactory.Start(ctx.Done())
//	boxStatefulSetInformerFactory.Start(ctx.Done())
//
//	<-ctx.Done()
//}

var _ = Describe("Test BoxStatefulSet Controller", func() {

	It("create BoxStatefulSet", func() {
		time.Sleep(5 * time.Second)
		ctx := context.Background()
		key := "chenkun/boxsts-test"
		err := controller.sync(ctx, key)
		Expect(err).Should(BeNil())

	})

	It("update BoxStatefulSet", func() {

	})

	It("delete BoxStatefulSet", func() {

	})
})
