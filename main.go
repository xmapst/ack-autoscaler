package main

import (
	"autoscaler/checker"
	"autoscaler/conf"
	_ "autoscaler/conf"
	"autoscaler/dataprovider"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	registerSignalHandlers()
}

func main() {
	logrus.Println("start...")
	// 带缓冲通道
	var dataCh = make(chan []*corev1.Pod, 100)
	// new一个数据提供者
	d := dataprovider.NewDataProvider(
		dataCh,
		conf.KubeCli, conf.Config.ReSync,
		conf.Config.TriggerTime,
		conf.Config.TriggerNo,
	)
	aliClient, err := conf.Config.CreateAliClient(tea.String(d.Region))
	if err != nil {
		logrus.Fatalln(err)
	}
	c := checker.NewChecker(aliClient, d.ClusterID)
	for pods := range dataCh {
		nodePoolID := d.GetNodePoolID(pods)
		if nodePoolID == "" {
			logrus.Error("cannot find the node pool ID, do nothing")
			continue
		}
		// 阻塞式检查集群状态是否适合扩容
		c.ClusterNodePoolState(nodePoolID)

		// 可操作时重新检查pending的pod
		pods = d.CheckNew(pods)
		// 查找本次操作的节点池
		nodePoolID = d.GetNodePoolID(pods)
		if nodePoolID == "" {
			logrus.Error("cannot find the node pool ID, do nothing")
			continue
		}
		logrus.Info("the node pool will be expanded soon ", nodePoolID)

		// 动态计算本次所需扩容的cpu/mem
		reqCpu, reqMem := d.GetNeededResources(pods)
		logrus.Printf("request cpu %d core mem %d M", reqCpu, reqMem)

		// 匹配响应规格以及算出需扩容的个数
		count := reqMem / conf.MEMStandard
		// 取余加1
		if reqMem%conf.MEMStandard > 0 {
			count += 1
		}
		logrus.Info("request node: ", count)
		// 进行扩容操作
		res, err := c.ScaleOutClusterNodePool(nodePoolID, count)
		if err != nil {
			logrus.Error(err)
			continue
		}
		logrus.Info("task id: ", *res.Body.TaskId)
		// 10分钟后检查本批次处理pod状态是否恢复正常, 否则丢回
		go d.RetryCheck(pods)
	}
}

func registerSignalHandlers() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigs
		logrus.Println("close!!!")
		os.Exit(0)
	}()
}
