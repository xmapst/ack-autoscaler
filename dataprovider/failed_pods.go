package dataprovider

import (
	corev1 "k8s.io/api/core/v1"
	"sync"
)

//FailedPod 表示一个调度失败的 pod
type FailedPod struct {
	Remediations int // 0:未处理, 1:处理中
	Pod          *corev1.Pod
}

//FailedPods 是未能调度的 pod 的集合
type FailedPods struct {
	failedPods map[string]*FailedPod
	lock       sync.Mutex
}

func newFailedDB() *FailedPods {
	return &FailedPods{
		failedPods: make(map[string]*FailedPod),
	}
}

func (f *FailedPods) addPod(name string, pod *corev1.Pod) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.failedPods[name] = &FailedPod{
		Remediations: 0,
		Pod:          pod,
	}
}

func (f *FailedPods) exist(name string) bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	_, ok := f.failedPods[name]
	return ok
}

func (f *FailedPods) total() int64 {
	f.lock.Lock()
	defer f.lock.Unlock()
	var count int64
	for _, v := range f.failedPods {
		if v.Remediations != 0 {
			continue
		}
		count++
	}
	return count
}

func (f *FailedPods) getPodByName(name string) *FailedPod {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.failedPods[name]
}

func (f *FailedPods) getPendingPods() []*corev1.Pod {
	f.lock.Lock()
	defer f.lock.Unlock()

	var pods []*corev1.Pod
	for _, p := range f.failedPods {
		if p.Remediations != 0 {
			continue
		}
		pods = append(pods, p.Pod)
	}
	return pods
}

func (f *FailedPods) setPodStateProcessing(name string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.failedPods[name].Remediations = 1
}

func (f *FailedPods) setPodStateUnProcessing(name string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.failedPods[name].Remediations = 0
}

func (f *FailedPods) removePod(name string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	delete(f.failedPods, name)
}
