package loadbalance

import (
	"context"
	"github.com/bytedance/gopkg/lang/fastrand"
	"github.com/zhihanii/discovery"
	"sync"
)

type weightedRoundRobinBalancer struct {
	pickerCache sync.Map
}

func NewWeightedRoundRobinBalancer() LoadBalancer {
	return &weightedRoundRobinBalancer{}
}

func (w *weightedRoundRobinBalancer) GetPicker(e discovery.Result) Picker {
	if !e.Cacheable {
		return w.createPicker(e)
	}

	picker, ok := w.pickerCache.Load(e.CacheKey)
	if !ok {
		picker = w.createPicker(e)
		w.pickerCache.Store(e.CacheKey, picker)
	}
	return picker.(Picker)
}

func (w *weightedRoundRobinBalancer) Name() string {
	return "weighted_round_robin"
}

func (w *weightedRoundRobinBalancer) Rebalance(e discovery.Result) {
	if !e.Cacheable {
		return
	}
	w.pickerCache.Store(e.CacheKey, w.createPicker(e))
}

func (w *weightedRoundRobinBalancer) Delete(e discovery.Result) {
	if !e.Cacheable {
		return
	}
	w.pickerCache.Delete(e.CacheKey)
}

func (w *weightedRoundRobinBalancer) createPicker(e discovery.Result) (picker Picker) {
	instances := make([]discovery.Instance, len(e.Instances))
	balance := true
	cnt := 0
	for _, instance := range e.Instances {
		weight := instance.Weight()
		if weight <= 0 {
			//log idx
			continue
		}
		instances[cnt] = instance
		if cnt > 0 && instances[cnt-1].Weight() != weight {
			balance = false
		}
		cnt++
	}
	instances = instances[:cnt]
	if len(instances) == 0 {
		return new(DummyPicker)
	}

	if balance { //所有Instance的权重都相同
		picker = newRoundRobinPicker(instances)
	} else {
		picker = newWeightedRoundRobinPicker(instances)
	}
	return picker
}

const wrrVNodesBatchSize = 500

type wrrNode struct {
	discovery.Instance
	current int
}

type weightedRoundRobinPicker struct {
	nodes []*wrrNode
	size  uint64

	iterator  *round
	vsize     uint64
	vcapacity uint64
	vnodes    []discovery.Instance
	vmux      sync.RWMutex
}

func newWeightedRoundRobinPicker(instances []discovery.Instance) Picker {
	w := &weightedRoundRobinPicker{
		size:     uint64(len(instances)),
		iterator: newRound(),
	}
	w.nodes = make([]*wrrNode, w.size)
	offset := fastrand.Uint64n(w.size)
	totalWeight := 0
	gcd := 0
	for idx := uint64(0); idx < w.size; idx++ {
		ins := instances[(idx+offset)%w.size]
		totalWeight += ins.Weight()
		gcd = gcdInt(gcd, ins.Weight())
		w.nodes[idx] = &wrrNode{
			Instance: ins,
			current:  0,
		}
	}

	w.vcapacity = uint64(totalWeight / gcd)
	w.vnodes = make([]discovery.Instance, w.vcapacity)
	w.buildVirtualWrrNodes(wrrVNodesBatchSize)
	return w
}

func (w *weightedRoundRobinPicker) Next(ctx context.Context, req any) (ins discovery.Instance) {
	idx := w.iterator.Next() % w.vcapacity
	w.vmux.RLock()
	ins = w.vnodes[idx]
	w.vmux.RUnlock()
	if ins != nil {
		return ins
	}

	w.vmux.Lock()
	defer w.vmux.Unlock()
	if w.vnodes[idx] != nil {
		return w.vnodes[idx]
	}
	vtarget := w.vsize + wrrVNodesBatchSize
	if idx >= vtarget {
		vtarget = idx + 1
	}
	w.buildVirtualWrrNodes(vtarget)
	return w.vnodes[idx]
}

func (w *weightedRoundRobinPicker) buildVirtualWrrNodes(vtarget uint64) {
	if vtarget > w.vcapacity {
		vtarget = w.vcapacity
	}
	for i := w.vsize; i < vtarget; i++ {
		w.vnodes[i] = nextWrrNode(w.nodes).Instance
	}
	w.vsize = vtarget
}

func nextWrrNode(nodes []*wrrNode) (selected *wrrNode) {
	maxCurrent := 0
	totalWeight := 0
	for _, node := range nodes {
		node.current += node.Weight()
		totalWeight += node.Weight()
		if selected == nil || node.current > maxCurrent {
			selected = node
			maxCurrent = node.current
		}
	}
	if selected == nil {
		return nil
	}
	selected.current -= totalWeight
	return selected
}

func gcdInt(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

type roundRobinPicker struct {
	size      uint64
	instances []discovery.Instance
	iterator  *round
}

func newRoundRobinPicker(instances []discovery.Instance) Picker {
	size := uint64(len(instances))
	return &roundRobinPicker{
		size:      size,
		instances: instances,
		iterator:  newRound(),
	}
}

func (r *roundRobinPicker) Next(ctx context.Context, req any) (ins discovery.Instance) {
	if r.size == 0 {
		return nil
	}
	idx := r.iterator.Next() % r.size
	ins = r.instances[idx]
	return
}
