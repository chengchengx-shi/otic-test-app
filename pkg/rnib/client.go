package rnib

import (
	"context"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
)

type Client struct {
	toposdk.Client
}

func NewClient() (*Client, error) {
	if cient, err := toposdk.NewClient(); err != nil {
		return nil, err
	} else {
		return &Client{cient}, nil
	}
}

func (c *Client) ListE2NodeIDs(ctx context.Context) ([]topoapi.ID, error) {
	filters := &topoapi.Filters{
		KindFilter: &topoapi.Filter{
			Filter: &topoapi.Filter_Equal_{
				Equal_: &topoapi.EqualFilter{
					Value: topoapi.CONTROLS,
				},
			},
		},
	}
	objects, err := c.Client.List(ctx, toposdk.WithListFilters(filters))
	if err != nil {
		return nil, err
	}

	e2NodeIDs := make([]topoapi.ID, 0, len(objects))
	for _, obj := range objects {
		relation := obj.GetRelation()
		e2NodeIDs = append(e2NodeIDs, relation.GetTgtEntityID())
	}
	return e2NodeIDs, nil
}

func (c *Client) GetE2Node(ctx context.Context, e2NodeID topoapi.ID) (*topoapi.E2Node, error) {
	var node topoapi.E2Node
	if obj, err := c.Client.Get(ctx, e2NodeID); err != nil {
		return nil, err
	} else {
		return &node, obj.GetAspect(&node)
	}
}

func (c *Client) ListE2NodeCells(ctx context.Context, e2NodeID topoapi.ID) ([]*topoapi.E2Cell, error) {
	filters := &topoapi.Filters{
		RelationFilter: &topoapi.RelationFilter{
			SrcId:        string(e2NodeID),
			RelationKind: topoapi.CONTAINS,
			TargetKind:   topoapi.E2CELL,
		},
	}
	objects, err := c.Client.List(ctx, toposdk.WithListFilters(filters))
	if err != nil {
		return nil, err
	}

	cells := make([]*topoapi.E2Cell, 0, len(objects))
	for _, obj := range objects {
		var cell topoapi.E2Cell
		if err := obj.GetAspect(&cell); err != nil {
			return nil, err
		}
		cells = append(cells, &cell)
	}
	return cells, nil
}
