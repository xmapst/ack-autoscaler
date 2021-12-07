package checker

import (
	cs "github.com/alibabacloud-go/cs-20151215/v2/client"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/sirupsen/logrus"
	"time"
)

type Checker struct {
	client    *cs.Client
	clusterId *string
}

func NewChecker(client *cs.Client, clusterId string) *Checker {
	return &Checker{
		client:    client,
		clusterId: tea.String(clusterId),
	}
}

func (c *Checker) ClusterState() bool {
	// 10秒检查一次集群健康状态
	for range time.Tick(10 * time.Second) {
		describeClusterDetailRes, _err := c.describeClusterDetail()
		if _err != nil {
			logrus.Error(_err)
			return false
		}

		if tea.BoolValue(util.EqualString(describeClusterDetailRes.Body.State, tea.String("running"))) {
			return true
		}
	}
	return false
}

func (c *Checker) ClusterNodePoolState(nodePoolId string) bool {
	// 10秒检查一次集群健康状态
	for range time.Tick(10 * time.Second) {
		describeClusterNodePoolDetailRes, _err := c.describeClusterNodePoolDetail(nodePoolId)
		if _err != nil {
			logrus.Error(_err)
			return false
		}
		if tea.BoolValue(util.EqualString(describeClusterNodePoolDetailRes.Body.Status.State, tea.String("active"))) {
			return true
		}
	}
	return false
}

/**
 * 查询集群状态
 */
func (c *Checker) describeClusterDetail() (_result *cs.DescribeClusterDetailResponse, _err error) {
	_result = &cs.DescribeClusterDetailResponse{}
	_body, _err := c.client.DescribeClusterDetail(c.clusterId)
	if _err != nil {
		return _result, _err
	}
	_result = _body
	return _result, _err
}

/**
 * 查询节点池状态
 */
func (c *Checker) describeClusterNodePoolDetail(nodePoolId string) (_result *cs.DescribeClusterNodePoolDetailResponse, _err error) {
	_result = &cs.DescribeClusterNodePoolDetailResponse{}
	_body, _err := c.client.DescribeClusterNodePoolDetail(c.clusterId, tea.String(nodePoolId))
	if _err != nil {
		return _result, _err
	}
	_result = _body
	return _result, _err
}

func (c *Checker) ScaleOutClusterNodePool(nodePoolId string, count int64) (_result *cs.ScaleClusterNodePoolResponse, _err error) {
	scaleClusterNodePoolRequest := &cs.ScaleClusterNodePoolRequest{
		Count: tea.Int64(count),
	}

	_body, _err := c.client.ScaleClusterNodePool(c.clusterId, tea.String(nodePoolId), scaleClusterNodePoolRequest)
	if _err != nil {
		return _result, _err
	}
	return _body, _err
}
