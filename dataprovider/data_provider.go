package dataprovider

import (
	"autoscaler/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"strings"
	"time"
)

type DataProvider struct {
	dataCh      chan<- []*corev1.Pod
	client      *kubernetes.Clientset
	factory     informers.SharedInformerFactory
	reSync      time.Duration
	triggerTime time.Duration
	triggerNo   int64
	FailedDB    *FailedPods
	stopCh      chan struct{}
	ClusterID   string
	Region      string
}

func NewDataProvider(dataCh chan<- []*corev1.Pod, client *kubernetes.Clientset, reSync, triggerTime time.Duration, triggerNo int64) *DataProvider {
	d := &DataProvider{
		dataCh:      dataCh,
		client:      client,
		reSync:      reSync,
		triggerTime: triggerTime,
		triggerNo:   triggerNo,
		stopCh:      make(chan struct{}),
		FailedDB:    newFailedDB(),
	}
	// informer的工厂函数,返回的是sharedInformerFactory对象
	d.factory = informers.NewSharedInformerFactory(d.client, d.reSync)
	go d.syncFailingPods()
	d.getClusterID()
	d.getRegion()
	d.export()
	return d
}

func (d *DataProvider) getClusterID() {
	clusterProfile, err := d.client.CoreV1().
		ConfigMaps(metav1.NamespaceSystem).
		Get(context.TODO(), "ack-cluster-profile", metav1.GetOptions{}) // 阿里云特有的cm ack-cluster-profile
	if err != nil {
		logrus.Fatalln(err)
	}
	var ok bool
	d.ClusterID, ok = clusterProfile.Data["clusterid"]
	if !ok {
		logrus.Fatalln("not found cluster id")
	}
	logrus.Info("cluster id ", d.ClusterID)
}

func (d *DataProvider) getRegion() {
	nodes, err := d.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{Limit: 1})
	if err != nil {
		logrus.Fatalln(err)
	}
	if len(nodes.Items) == 0 {
		logrus.Fatalln("not found region")
	}
	var ok bool
	d.Region, ok = nodes.Items[0].Labels["topology.kubernetes.io/region"]
	if !ok {
		logrus.Fatalln("not found region")
	}
	logrus.Info("cluster region ", d.Region)
}

// syncFailingPods 更新 DataProvider 的 FailedPods, 使用ListerWatcher接口, 默认30s全量同步
func (d *DataProvider) syncFailingPods() {
	defer close(d.stopCh)
	// 创建Pod资源对象的Informer
	informer := d.factory.Core().V1().Pods().Informer()
	// 注册事件回调函数
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    d.addFunc,
		UpdateFunc: d.updateFunc,
		DeleteFunc: d.deleteFunc,
	})
	informer.Run(d.stopCh)
}

func (d *DataProvider) updateFunc(_, newObj interface{}) {
	newPod := newObj.(*corev1.Pod)
	if newPod.Status.Phase != corev1.PodPending {
		d.deleteFunc(newObj)
	} else {
		d.addFunc(newObj)
	}
}

func (d *DataProvider) addFunc(obj interface{}) {
	pod := obj.(*corev1.Pod)
	if len(pod.Status.Conditions) < 1 {
		return
	}
	condition := pod.Status.Conditions[0]
	if condition.Reason != "Unschedulable" {
		return
	}
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	// 避免更新操作中的
	if d.FailedDB.exist(key) {
		return
	}
	msg := condition.Message
	if strings.Contains(msg, "Insufficient cpu") ||
		strings.Contains(msg, "Insufficient memory") {
		logrus.Debug("add pod ", key)
		d.FailedDB.addPod(key, pod)
	}
}

func (d *DataProvider) deleteFunc(obj interface{}) {
	pod := obj.(*corev1.Pod)
	if len(pod.Status.Conditions) < 1 {
		return
	}
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	if !d.FailedDB.exist(key) {
		return
	}
	logrus.Debug("del pod ", key)
	d.FailedDB.removePod(key)
}

func (d *DataProvider) GetNodePoolID(pods []*corev1.Pod) string {
	var nodePoolID string
	for _, pod := range pods {
		label := labels.SelectorFromSet(pod.Spec.NodeSelector)
		nodes, err := d.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
			LabelSelector: label.String(),
			Limit:         1,
		})
		if err != nil {
			logrus.Warning(err)
			continue
		}
		if len(nodes.Items) == 0 {
			logrus.Warning("not found node")
			continue
		}
		nodePoolID = nodes.Items[0].Labels["alibabacloud.com/nodepool-id"]
		if nodePoolID != "" {
			break
		}
	}
	return nodePoolID
}

func (d *DataProvider) CheckNew(oldPods []*corev1.Pod) (pods []*corev1.Pod) {
	for _, pod := range oldPods {
		name, _ := cache.MetaNamespaceKeyFunc(pod)
		if !d.FailedDB.exist(name) {
			continue
		}
		pods = append(pods, pod)
	}
	return
}

func (d *DataProvider) RetryCheck(pods []*corev1.Pod) {
	<-time.Tick(10 * time.Minute)
	for _, pod := range pods {
		name, _ := cache.MetaNamespaceKeyFunc(pod)
		if d.FailedDB.exist(name) {
			d.FailedDB.setPodStateUnProcessing(name)
		}
	}
}

func (d *DataProvider) GetNeededResources(pods []*corev1.Pod) (cpuTotal, memTotal int64) {
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			cpuTotal += utils.GetResourceCPU(container.Resources)
			memTotal += utils.GetResourceMem(container.Resources)
		}
	}
	return
}

func (d *DataProvider) export() {
	var countCh = make(chan bool)
	go d.totalTrigger(countCh)
	go func() {
		for {
			select {
			case <-countCh:
				// 量到时间没到
				d.get()
			case <-time.Tick(d.triggerTime):
				// 时间到量不到
				d.get()
			}
		}
	}()
}

func (d *DataProvider) totalTrigger(countCh chan bool) {
	for {
		if d.FailedDB.total() >= d.triggerNo {
			countCh <- true
		}
		// 适当睡眠, 减少cpu使用
		time.Sleep(100 * time.Millisecond)
	}
}

func (d *DataProvider) get() {
	if d.FailedDB.total() <= 0 {
		return
	}
	// 取出此时所有待处理的pod
	pendingPods := d.FailedDB.getPendingPods()
	for _, pod := range pendingPods {
		key, _ := cache.MetaNamespaceKeyFunc(pod)
		d.FailedDB.setPodStateProcessing(key)
		logrus.Info("enter the processing queue ", key)
	}
	d.dataCh <- pendingPods
}
