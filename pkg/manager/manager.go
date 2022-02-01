// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package manager

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-kpimon/pkg/broker"
	appConfig "github.com/onosproject/onos-kpimon/pkg/config"
	nbi "github.com/onosproject/onos-kpimon/pkg/northbound"
	"github.com/onosproject/onos-kpimon/pkg/northbound/a1"
	"github.com/onosproject/onos-kpimon/pkg/southbound/e2/subscription"
	"github.com/onosproject/onos-kpimon/pkg/store/actions"
	"github.com/onosproject/onos-kpimon/pkg/store/measurements"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
)

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	CAPath      string
	KeyPath     string
	CertPath    string
	E2tEndpoint string
	GRPCPort    int
	RicActionID int32
	ConfigPath  string
	SMName      string
	SMVersion   string
}

// NewManager generates the new KPIMON xAPP manager
func NewManager(config Config) *Manager {
	appCfg, err := appConfig.NewConfig(config.ConfigPath)
	if err != nil {
		log.Warn(err)
	}
	subscriptionBroker := broker.NewBroker()
	measStore := measurements.NewStore()
	actionsStore := actions.NewStore()

	subManager, err := subscription.NewManager(
		subscription.WithE2TAddress("onos-e2t", 5150),
		subscription.WithServiceModel(subscription.ServiceModelName(config.SMName),
			subscription.ServiceModelVersion(config.SMVersion)),
		subscription.WithAppConfig(appCfg),
		subscription.WithAppID("onos-kpimon"),
		subscription.WithBroker(subscriptionBroker),
		subscription.WithActionStore(actionsStore),
		subscription.WithMeasurementStore(measStore))

	if err != nil {
		log.Warn(err)
	}

	a1PolicyTypes := make([]*topo.A1PolicyType, 0)
	a1Policy := &topo.A1PolicyType{
		Name: "ORAN_TrafficSteeringPreference",
		Version: "2.0.0",
		ID: "ORAN_TrafficSteeringPreference_2.0.0",
		Description: "O-RAN traffic steering",
	}
	a1PolicyTypes = append(a1PolicyTypes, a1Policy)

	a1Manager, err := a1.NewManager(config.CAPath, config.KeyPath, config.CertPath, config.GRPCPort, "onos-kpimon", a1PolicyTypes)
	if err != nil {
		log.Warn(err)
	}

	manager := &Manager{
		appConfig:        appCfg,
		config:           config,
		subManager:       subManager,
		measurementStore: measStore,
		a1Manager: *a1Manager,
	}
	return manager
}

// Manager is an abstract struct for manager
type Manager struct {
	appConfig        appConfig.Config
	config           Config
	measurementStore measurements.Store
	subManager       subscription.Manager
	a1Manager        a1.Manager
}

// Run runs KPIMON manager
func (m *Manager) Run() {
	err := m.start()
	if err != nil {
		log.Errorf("Error when starting KPIMON: %v", err)
	}
}

// Close closes manager
func (m *Manager) Close() {
	log.Info("closing Manager")
	m.a1Manager.Close(context.Background())
}

func (m *Manager) start() error {
	err := m.startNorthboundServer()
	if err != nil {
		log.Warn(err)
		return err
	}

	err = m.subManager.Start()
	if err != nil {
		log.Warn(err)
		return err
	}

	m.a1Manager.Start()

	return nil
}

func (m *Manager) startNorthboundServer() error {
	s := northbound.NewServer(northbound.NewServerCfg(
		m.config.CAPath,
		m.config.KeyPath,
		m.config.CertPath,
		int16(m.config.GRPCPort),
		true,
		northbound.SecurityConfig{}))

	s.AddService(nbi.NewService(m.measurementStore))
	s.AddService(a1.NewA1EIService())
	s.AddService(a1.NewA1PService())

	doneCh := make(chan error)
	go func() {
		err := s.Serve(func(started string) {
			log.Info("Started NBI on ", started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}

// GetMeasurementStore returns measurement store
func (m *Manager) GetMeasurementStore() measurements.Store {
	return m.measurementStore
}
