package loadbalance

import (
	"context"
	"github.com/zhihanii/discovery"
	"github.com/zhihanii/registry"
	"sync"
	"sync/atomic"
)

type LoadBalancerBuildFunc func() LoadBalancer

type BalancerFactory struct {
	cache           sync.Map
	lbBuildFunc     LoadBalancerBuildFunc
	resolverBuilder registry.ResolverBuilder
}

func NewBalancerFactory(lbBuildFunc LoadBalancerBuildFunc, resolverBuilder registry.ResolverBuilder) *BalancerFactory {
	return &BalancerFactory{
		lbBuildFunc:     lbBuildFunc,
		resolverBuilder: resolverBuilder,
	}
}

func (f *BalancerFactory) BuildBalancers(ctx context.Context, targets []string) error {
	var err error
	var balancers []*Balancer
	for _, target := range targets {
		var b *Balancer
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

func (f *BalancerFactory) Get(ctx context.Context, target string) (*Balancer, error) {
	v, ok := f.cache.Load(target)
	if ok {
		return v.(*Balancer), nil
	}

	b, err := f.buildBalancer(target)
	if err != nil {
		return nil, err
	}
	f.cache.Store(target, b)
	return b, nil
}

func (f *BalancerFactory) buildBalancer(target string) (*Balancer, error) {
	var err error
	b := &Balancer{
		m:      f,
		lb:     f.lbBuildFunc(),
		target: target,
	}
	b.resolver, err = f.resolverBuilder.Build(target, func(e discovery.Result) error {
		b.SetResult(e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

type Balancer struct {
	m  *BalancerFactory
	lb LoadBalancer
	//resolver discovery.Resolver
	resolver registry.Resolver
	target   string
	res      atomic.Value
	expire   int32
}

func (b *Balancer) GetPicker() Picker {
	res := b.res.Load().(discovery.Result)
	return b.lb.GetPicker(res)
}

func (b *Balancer) SetResult(res discovery.Result) {
	b.res.Store(res)
}
