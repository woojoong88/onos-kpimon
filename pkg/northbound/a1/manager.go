// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package a1

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/a1/connection"
)

var log = logging.GetLogger("a1")

func NewManager(caPath string, keyPath string, certPath string, grpcPort int, xAppName string, a1PolicyTypes []*topo.A1PolicyType) (*Manager, error) {
	a1ConnManager, err := a1connection.NewManager(caPath, keyPath, certPath, grpcPort, a1PolicyTypes)
	if err != nil {
		return nil, err
	}
	return &Manager{
		a1ConnManager: a1ConnManager,
	}, nil
}

type Manager struct {
	a1ConnManager *a1connection.Manager
}

func (m *Manager) Start() {
	m.a1ConnManager.Start(context.Background())
}

func (m *Manager) Close(ctx context.Context) {
	err := m.a1ConnManager.DeleteXAppElementOnTopo(ctx)
	if err != nil {
		log.Error(err)
	}
}