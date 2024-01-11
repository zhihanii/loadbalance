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
	//Rebalance(discovery.Result)
	//Delete(discovery.Result)
}

//type Rebalancer interface {
//	Rebalance(discovery.Change)
//	Delete(discovery.Change)
//}
