package kpm

import (
	"context"
	"fmt"
	"sort"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2_go/pdubuilder"
	e2smkpmv2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2_go/v2/e2sm-kpm-v2-go"
	e2sdk "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"
	"google.golang.org/protobuf/proto"
)

const (
	ServiceModelOID     = "1.3.6.1.4.1.53148.1.2.2.2"
	ServiceModelName    = "oran-e2sm-kpm"
	ServiceModelVersion = "v2"

	ReportPeriodInterval    = 1000
	ReportPeriodGranularity = 1000
)

type Indication struct {
	Header  *e2smkpmv2.E2SmKpmIndicationHeader
	Message *e2smkpmv2.E2SmKpmIndicationMessage
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

func (c *Client) Subscribe(ctx context.Context, e2NodeID topoapi.ID, reportStyle *topoapi.KPMReportStyle, cells []*topoapi.E2Cell) (<-chan e2api.Indication, error) {
	triggerPayload, err := createEventTriggerDefinitionPayload(ReportPeriodInterval)
	if err != nil {
		return nil, err
	}

	actions, err := createSubscriptionActions(reportStyle, cells)
	if err != nil {
		return nil, err
	}

	node := e2sdk.NodeID(e2NodeID)
	subName := fmt.Sprintf("kpm-%v-%v", e2NodeID, reportStyle.Type)
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
	var header e2smkpmv2.E2SmKpmIndicationHeader
	if err := proto.Unmarshal(indication.Header, &header); err != nil {
		return nil, err
	}

	var message e2smkpmv2.E2SmKpmIndicationMessage
	if err := proto.Unmarshal(indication.Payload, &message); err != nil {
		return nil, err
	}

	return &Indication{Header: &header, Message: &message}, nil
}

func createEventTriggerDefinitionPayload(period int64) ([]byte, error) {
	definition, err := pdubuilder.CreateE2SmKpmEventTriggerDefinition(period)
	if err != nil {
		return nil, err
	}
	if err := definition.Validate(); err != nil {
		return nil, err
	}
	return proto.Marshal(definition)
}

func createSubscriptionActions(reportStyle *topoapi.KPMReportStyle, cells []*topoapi.E2Cell) ([]e2api.Action, error) {
	sort.Slice(cells, func(i, j int) bool {
		return cells[i].CellObjectID < cells[j].CellObjectID
	})

	actions := make([]e2api.Action, 0, len(cells))
	for i, cell := range cells {
		payload, err := createActionDefinitionPayload(int64(i+1), cell.CellObjectID, reportStyle)
		if err != nil {
			return nil, err
		}

		action := &e2api.Action{
			ID:   int32(i),
			Type: e2api.ActionType_ACTION_TYPE_REPORT,
			SubsequentAction: &e2api.SubsequentAction{
				Type:       e2api.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
				TimeToWait: e2api.TimeToWait_TIME_TO_WAIT_ZERO,
			},
			Payload: payload,
		}
		actions = append(actions, *action)
	}
	return actions, nil
}

func createActionDefinitionPayload(subID int64, cellObjectID string, reportStyle *topoapi.KPMReportStyle) ([]byte, error) {
	measInfoList := &e2smkpmv2.MeasurementInfoList{
		Value: make([]*e2smkpmv2.MeasurementInfoItem, 0, len(reportStyle.Measurements)),
	}
	for _, measurement := range reportStyle.Measurements {
		infoItem, err := createMeasurementInfoItem(measurement)
		if err != nil {
			return nil, err
		}
		measInfoList.Value = append(measInfoList.Value, infoItem)
	}

	actionDefinition, err := pdubuilder.CreateActionDefinitionFormat1(cellObjectID, measInfoList, ReportPeriodGranularity, subID)
	if err != nil {
		return nil, err
	}

	kpmActionDefinition, err := pdubuilder.CreateE2SmKpmActionDefinitionFormat1(reportStyle.Type, actionDefinition)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(kpmActionDefinition)
}

func createMeasurementInfoItem(measurement *topoapi.KPMMeasurement) (*e2smkpmv2.MeasurementInfoItem, error) {
	typeName, err := pdubuilder.CreateMeasurementTypeMeasName(measurement.Name)
	if err != nil {
		return nil, err
	}
	return pdubuilder.CreateMeasurementInfoItem(typeName)
}
