package loadbalance

import (
	"context"
	"github.com/zhihanii/discovery"
)

var _ Picker = (*DummyPicker)(nil)

type DummyPicker struct{}

func (d *DummyPicker) Next(ctx context.Context, req any) (ins discovery.Instance) {
	return nil
}
