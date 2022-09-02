package kpm_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	prototypes "github.com/gogo/protobuf/types"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	e2sdk "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"
	"github.com/stretchr/testify/suite"

	"otic-test-app/pkg/kpm"
	"otic-test-app/pkg/rnib"
)

type TestContext struct {
	E2NodeIDs []topoapi.ID
	Functions []*topoapi.KPMRanFunction
}

type TestSuite struct {
	suite.Suite
	TestContext
	E2Client   *kpm.Client
	RnibClient *rnib.Client
}

func (ts *TestSuite) SetupSuite() {
	ts.E2Client = kpm.NewClient()

	var err error
	if ts.RnibClient, err = rnib.NewClient(); err != nil {
		ts.FailNow(err.Error(), "init topo client")
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	e2NodeIDs, err := ts.RnibClient.ListE2NodeIDs(ctx)
	if err != nil {
		ts.FailNow(err.Error(), "list e2nodes")
	}

	ts.TestContext.E2NodeIDs = e2NodeIDs
	ts.T().Logf("e2 node ids: %v", e2NodeIDs)
}

func (ts *TestSuite) TearDownSuite() {
}

func (ts *TestSuite) TestKPM() {
	ts.testServiceModel()
	ts.testSubscribe()
}

func (ts *TestSuite) testServiceModel() {
	for _, nodeID := range ts.TestContext.E2NodeIDs {
		ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
		defer cancel()

		node, err := ts.RnibClient.GetE2Node(ctx, nodeID)
		if err != nil {
			ts.FailNow(err.Error(), "get e2node %v", nodeID)
		}

		for _, sm := range node.GetServiceModels() {
			if sm.OID != kpm.ServiceModelOID {
				continue
			}
			ts.TestContext.Functions = make([]*topoapi.KPMRanFunction, 0, len(sm.RanFunctions))

			for i, anyObj := range sm.RanFunctions {
				var function topoapi.KPMRanFunction
				if err := prototypes.UnmarshalAny(anyObj, &function); err != nil {
					ts.Errorf(err, "unmarshal kpm function")
				}
				ts.TestContext.Functions = append(ts.TestContext.Functions, &function)

				for _, style := range function.ReportStyles {
					ts.T().Logf("e2node[%v].model[%v].function[%v].style[%v]: %v",
						nodeID, sm.Name, sm.RanFunctionIDs[i], style.Type, style.Name)
				}
			}
		}
	}
}

func (ts *TestSuite) testSubscribe() {
	for _, e2NodeID := range ts.TestContext.E2NodeIDs {
		rpcctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
		defer cancel()

		cells, err := ts.RnibClient.ListE2NodeCells(rpcctx, e2NodeID)
		if err != nil {
			ts.FailNow(err.Error(), "list cell ids on e2node %v", e2NodeID)
		}

		var wg sync.WaitGroup
		for _, function := range ts.TestContext.Functions {
			for _, reportStyle := range function.ReportStyles {
				wg.Add(1)

				go func(wg *sync.WaitGroup, e2NodeID topoapi.ID, reportStyle *topoapi.KPMReportStyle, cells []*topoapi.E2Cell) {
					defer wg.Done()

					subctx, cancel := context.WithCancel(context.TODO())
					defer cancel()

					subName := fmt.Sprintf("kpm-%v-%v", e2NodeID, reportStyle.Type)
					if indCh, err := ts.E2Client.Subscribe(subctx, e2NodeID, reportStyle, cells); err != nil {
						ts.FailNow(err.Error(), "subscription: %v", subName)
					} else {
						ts.T().Logf("subscription %v", subName)
						for i := 0; i < 3; i++ {
							indication := <-indCh
							if ind, err := kpm.ProcessIndication(&indication); err != nil {
								ts.FailNow(err.Error(), "process indication")
							} else {
								var (
									header  = ind.Header.GetIndicationHeaderFormats().GetIndicationHeaderFormat1()
									message = ind.Message.GetIndicationMessageFormats().GetIndicationMessageFormat1()
								)
								ts.T().Logf("header: %v", header)
								ts.T().Logf("message: %v", message)
							}

						}

						node := e2sdk.NodeID(e2NodeID)
						if err := ts.E2Client.Node(node).Unsubscribe(subctx, subName); err != nil {
							ts.FailNow(err.Error(), "unsubscribe %v", subName)
						}
						ts.T().Logf("unsubscribe %v", subName)
					}
				}(&wg, e2NodeID, reportStyle, cells)
			}
		}
		wg.Wait()
	}
}

func TestKPM(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
