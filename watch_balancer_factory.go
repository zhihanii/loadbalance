package loadbalance

import (
	"context"
	"github.com/zhihanii/discovery"
	"github.com/zhihanii/registry"
	"sync"
	"sync/atomic"
)

type WatchBalancerFactory struct {
	cache                sync.Map
	lbBuildFunc          LoadBalancerBuildFunc
	watchResolverBuilder registry.WatchResolverBuilder
}

func NewWatchBalancerFactory(lbBuildFunc LoadBalancerBuildFunc, watchResolverBuilder registry.WatchResolverBuilder) *WatchBalancerFactory {
	return &WatchBalancerFactory{
		lbBuildFunc:          lbBuildFunc,
		watchResolverBuilder: watchResolverBuilder,
	}
}

func (f *WatchBalancerFactory) BuildBalancers(ctx context.Context, targets []string) error {
	var err error
	var balancers []*WatchBalancer
	for _, target := range targets {
		var b *WatchBalancer
		b, err = f.buildBalancer(target)
		if err != nil {
			return err
		}
		balancers = append(balancers, b)
	}
	for _, b := range balancers {
		f.cache.Store(b.target, b)
	}
	return nil
}

func (f *WatchBalancerFactory) Get(ctx context.Context, target string) (*WatchBalancer, error) {
	v, ok := f.cache.Load(target)
	if ok {
		return v.(*WatchBalancer), nil
	}
	b, err := f.buildBalancer(target)
	if err != nil {
		return nil, err
	}
	f.cache.Store(target, b)
	return b, nil
}

func (f *WatchBalancerFactory) buildBalancer(target string) (*WatchBalancer, error) {
	var err error
	b := &WatchBalancer{
		f:      f,
		lb:     f.lbBuildFunc(),
		target: target,
	}
	b.watchResolver, err = f.watchResolverBuilder.Build(target, func(e discovery.Result) error {
		b.SetResult(e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

type WatchBalancer struct {
	f             *WatchBalancerFactory
	lb            LoadBalancer
	watchResolver registry.WatchResolver
	target        string
	result        atomic.Value
	expire        int32
}

func (b *WatchBalancer) GetPicker() Picker {
	e := b.result.Load().(discovery.Result)
	return b.lb.GetPicker(e)
}

func (b *WatchBalancer) SetResult(e discovery.Result) {
	b.result.Store(e)
}
