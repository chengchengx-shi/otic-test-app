package rc_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	e2sdk "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"
	"github.com/stretchr/testify/suite"

	"otic-test-app/pkg/rc"
	"otic-test-app/pkg/rnib"
)

type TestContext struct {
	E2NodeIDs []topoapi.ID
}

type TestSuite struct {
	suite.Suite
	TestContext
	E2Client   *rc.Client
	RnibClient *rnib.Client
}

func (ts *TestSuite) SetupSuite() {
	ts.E2Client = rc.NewClient()

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

func (ts *TestSuite) TestRC() {
	// ts.testServiceModel()
	ts.testSubscribe()
}

// func (ts *TestSuite) testServiceModel() {
// 	for _, nodeID := range ts.TestContext.E2NodeIDs {
// 		ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
// 		defer cancel()

// 		node, err := ts.RnibClient.GetE2Node(ctx, nodeID)
// 		if err != nil {
// 			ts.FailNow(err.Error(), "get e2node %v", nodeID)
// 		}

// 		for _, sm := range node.GetServiceModels() {
// 			if sm.OID != rc.ServiceModelOID {
// 				continue
// 			}
// 		}
// 	}
// }

func (ts *TestSuite) testSubscribe() {
	var wg sync.WaitGroup
	for _, e2NodeID := range ts.TestContext.E2NodeIDs {
		wg.Add(1)

		go func(wg *sync.WaitGroup, e2NodeID topoapi.ID) {
			defer wg.Done()

			subctx, cancel := context.WithCancel(context.TODO())
			defer cancel()

			subName := fmt.Sprintf("rc-%v-sinr", e2NodeID)
			if indCh, err := ts.E2Client.Subscribe(subctx, e2NodeID); err != nil {
				ts.FailNow(err.Error(), "subscription: %v", subName)
			} else {
				ts.T().Logf("subscription %v", subName)
				for i := 0; i < 1; i++ {
					indication := <-indCh
					if ind, err := rc.ProcessIndication(&indication); err != nil {
						ts.FailNow(err.Error(), "process indication")
					} else {
						var (
							header  = ind.Header.GetRicIndicationHeaderFormats().GetIndicationHeaderFormat1()
							message = ind.Message.GetRicIndicationMessageFormats().GetIndicationMessageFormat3()
						)
						ts.T().Logf("e2node %v condition id: %v", e2NodeID, header.GetRicEventTriggerConditionId().GetValue())
						for _, cellInfo := range message.GetCellInfoList() {
							nrcID := cellInfo.GetCellGlobalId().GetNRCgi().GetNRcellIdentity()
							arfcn := cellInfo.GetNeighborRelationTable().GetServingCellArfcn().GetNR()
							ts.T().Logf("e2node %v cell %v arfcn %v", e2NodeID, nrcID.GetValue().GetValue(), arfcn.GetNRarfcn())
						}
					}
				}

				node := e2sdk.NodeID(e2NodeID)
				if err := ts.E2Client.Node(node).Unsubscribe(subctx, subName); err != nil {
					ts.FailNow(err.Error(), "unsubscribe %v", subName)
				}
				ts.T().Logf("unsubscribe %v", subName)
			}
		}(&wg, e2NodeID)
	}
	wg.Wait()
}

func TestRC(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
