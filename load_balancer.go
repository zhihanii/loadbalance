package loadbalance

import (
	"context"
	"github.com/zhihanii/discovery"
)

type Picker interface {
	Next(ctx context.Context, req any) discovery.Instance
}

type LoadBalancer interface {
	GetPicker(discovery.Result) Picker
	Name() string
}

//type Rebalancer interface {
//	Rebalance(discovery.Change)
//	Delete(discovery.Change)
//}
