package conf

import (
	"autoscaler/utils"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"log"
	"time"
)

var (
	Config      Configure
	MEMStandard int64
)

type Configure struct {
	LogLevel        string        `json:"LOG_LEVEL" envconfig:"LOG_LEVEL" default:"DEBUG"`                                          // 日志等级
	KubeConf        string        `json:"KUBECONFIG" envconfig:"KUBECONFIG"`                                                        // 用于本地测试
	ReSync          time.Duration `json:"RE_SYNC" envconfig:"RE_SYNC" default:"30s"`                                                // 全量同步时间间隔
	TriggerTime     time.Duration `json:"TRIGGER_TIME" envconfig:"TRIGGER_TIME" default:"1m"`                                       // 触发时间
	TriggerNo       int64         `json:"TRIGGER_NO" envconfig:"TRIGGER_NO" default:"10"`                                           // 触发数量
	AccessKeyId     string        `json:"ACCESS_KEY_ID" envconfig:"ACCESS_KEY_ID" required:"true"`                                  // 阿里云accessKeyId
	AccessKeySecret string        `json:"ACCESS_KEY_SECRET" envconfig:"ACCESS_KEY_SECRET" required:"true"`                          // 阿里云accessKeySecret
	MemoryStandard  string        `json:"MEMORY_STANDARD" envconfig:"MEMORY_STANDARD" default:"28Gi"`                               // 内存标准, 用于计算节点个数
}

func init() {
	logrus.SetFormatter(&utils.ConsoleFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)

	// load the configuration from the environment.
	err := envconfig.Process("", &Config)
	if err != nil {
		log.Println("配置加载错误")
		log.Fatal(err)
	}
	logLevel, err := logrus.ParseLevel(Config.LogLevel)
	if err != nil {
		log.Fatalln(err)
	}
	logrus.SetLevel(logLevel)
	MEMStandard, err = utils.ParseMemory(Config.MemoryStandard)
	if err != nil {
		log.Fatalln("MEMORY_STANDARD", err)
	}
	Config.initKube()
}
