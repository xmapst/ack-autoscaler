package utils

import (
	"errors"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// GetResourceMem bytes
func GetResourceMem(mem coreV1.ResourceRequirements) int64 {
	if (*mem.Limits.Cpu() != resource.Quantity{} && mem.Limits.Memory().Value() > 0) {
		return mem.Limits.Memory().Value()
	}
	return mem.Requests.Memory().Value()
}

func GetResourceCPU(cpu coreV1.ResourceRequirements) int64 {
	if (*cpu.Limits.Cpu() != resource.Quantity{} && cpu.Limits.Cpu().MilliValue() > 0) {
		return cpu.Limits.Cpu().MilliValue() / 1000
	}
	return cpu.Requests.Cpu().MilliValue() / 1000
}

//ParseMemory bytes
func ParseMemory(mem string) (int64, error) {
	q, err := resource.ParseQuantity(mem)
	if err != nil {
		return 0, err
	}
	n, ok := q.AsInt64()
	if !ok {
		return 0, errors.New("parse memory failed")
	}
	return n, nil
}
