package conf

import (
	cs "github.com/alibabacloud-go/cs-20151215/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeCertification *rest.Config
	KubeCli           *kubernetes.Clientset
)

func (c *Configure) initKube() {
	c.initKubeConf()
	c.initKubeCli()
}

func (c *Configure) initKubeCli() {
	var err error
	KubeCli, err = kubernetes.NewForConfig(kubeCertification)
	if err != nil {
		logrus.Fatal(err)
	}
}

func (c *Configure) initKubeConf() {
	var err error
	if c.KubeConf != "" {
		bs, err := ioutil.ReadFile(c.KubeConf)
		if err != nil {
			logrus.Fatal(err)
		}
		kubeCertification, err = clientcmd.RESTConfigFromKubeConfig(bs)
	} else {
		kubeCertification, err = rest.InClusterConfig()
	}
	if err != nil {
		logrus.Fatal(err)
	}
}

// CreateAliClient
/**
 * 使用AK&SK初始化账号Client
 * @param accessKeyId
 * @param accessKeySecret
 * @param regionId
 * @param endpoint
 * @return Client
 * @throws Exception
 */
func (c *Configure) CreateAliClient(regionId *string) (_result *cs.Client, _err error) {
	config := &openapi.Config{}
	// 您的AccessKey ID
	config.AccessKeyId = tea.String(c.AccessKeyId)
	// 您的AccessKey Secret
	config.AccessKeySecret = tea.String(c.AccessKeySecret)
	// 您的可用区ID
	config.RegionId = regionId
	_result = &cs.Client{}
	_result, _err = cs.NewClient(config)
	return _result, _err
}
