package rc

import (
	"context"
	"fmt"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/pdubuilder"
	e2smrc "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/v1/e2sm-rc-ies"
	e2sdk "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"
	"google.golang.org/protobuf/proto"
)

const (
	ServiceModelOID     = "1.3.6.1.4.1.53148.1.1.2.3"
	ServiceModelName    = "oran-e2sm-rc"
	ServiceModelVersion = "v1"

	// E2Node Information Change
	CellConfigurationChange    int32 = 1
	CellNeighborRelationChange int32 = 2

	E2NodeInformationStyle int32 = 3
	ParameterSINR          int64 = 12503
)

type Indication struct {
	Header  *e2smrc.E2SmRcIndicationHeader
	Message *e2smrc.E2SmRcIndicationMessage
}

type Client struct {
	e2sdk.Client
}

func NewClient() *Client {
	client := e2sdk.NewClient(
		e2sdk.WithServiceModel(ServiceModelName, ServiceModelVersion),
	)
	return &Client{Client: client}
}

func (c *Client) Subscribe(ctx context.Context, e2NodeID topoapi.ID) (<-chan e2api.Indication, error) {
	triggerPayload, err := createEventTriggerDefinitionPayload()
	if err != nil {
		return nil, err
	}
	actions, err := createSubscriptionActions()
	if err != nil {
		return nil, err
	}

	node := e2sdk.NodeID(e2NodeID)
	subName := fmt.Sprintf("rc-%v-sinr", e2NodeID)
	subSpec := e2api.SubscriptionSpec{
		Actions: actions,
		EventTrigger: e2api.EventTrigger{
			Payload: triggerPayload,
		},
	}
	indCh := make(chan e2api.Indication)
	_, err = c.Client.Node(node).Subscribe(ctx, subName, subSpec, indCh)
	if err != nil {
		return nil, err
	}
	return indCh, nil
}

func ProcessIndication(indication *e2api.Indication) (*Indication, error) {
	var header e2smrc.E2SmRcIndicationHeader
	if err := proto.Unmarshal(indication.Header, &header); err != nil {
		return nil, err
	}

	var message e2smrc.E2SmRcIndicationMessage
	if err := proto.Unmarshal(indication.Payload, &message); err != nil {
		return nil, err
	}

	return &Indication{Header: &header, Message: &message}, nil
}

func createEventTriggerDefinitionPayload() ([]byte, error) {
	item1, err := pdubuilder.CreateE2SmRcEventTriggerFormat3Item(CellConfigurationChange, CellConfigurationChange)
	if err != nil {
		return nil, err
	}
	item2, err := pdubuilder.CreateE2SmRcEventTriggerFormat3Item(CellNeighborRelationChange, CellNeighborRelationChange)
	if err != nil {
		return nil, err
	}

	items := []*e2smrc.E2SmRcEventTriggerFormat3Item{item1, item2}
	trigger, err := pdubuilder.CreateE2SmRcEventTriggerFormat3(items)
	if err != nil {
		return nil, err
	}
	if err := trigger.Validate(); err != nil {
		return nil, err
	}
	return proto.Marshal(trigger)
}

func createSubscriptionActions() ([]e2api.Action, error) {
	parameters := []int64{ParameterSINR}
	definition, err := pdubuilder.CreateE2SmRcActionDefinitionFormat1(E2NodeInformationStyle, parameters)
	if err != nil {
		return nil, err
	}
	if err := definition.Validate(); err != nil {
		return nil, err
	}

	payload, err := proto.Marshal(definition)
	if err != nil {
		return nil, err
	}

	action := &e2api.Action{
		ID:      int32(3),
		Type:    e2api.ActionType_ACTION_TYPE_REPORT,
		Payload: payload,
	}
	return []e2api.Action{*action}, nil
}
