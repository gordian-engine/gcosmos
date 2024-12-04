package gcapp

import (
	"fmt"
	"slices"

	"cosmossdk.io/core/registry"
	"cosmossdk.io/core/transaction"
	"cosmossdk.io/depinject"
	clog "cosmossdk.io/log"
	"cosmossdk.io/runtime/v2"
	serverstore "cosmossdk.io/server/v2/store"
	"cosmossdk.io/store/v2"
	"cosmossdk.io/store/v2/root"
	basedepinject "cosmossdk.io/x/accounts/defaults/base/depinject"
	stakingkeeper "cosmossdk.io/x/staking/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
)

type GCApp struct {
	*runtime.App[transaction.Tx] // Not yet sure why this is embedded, but that's what simapp does.

	legacyAmino       registry.AminoRegistrar
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry
	store             store.RootStore

	// required keepers during wiring
	// others keepers are all in the app
	UpgradeKeeper *upgradekeeper.Keeper
	StakingKeeper *stakingkeeper.Keeper
}

// AppConfig is roughly equivalent to the simapp AppConfig function.
// It needs to be exported for use in a client-only config,
// when running a clientside-only gcosmos command.
func AppConfig() depinject.Config {
	return depinject.Configs(
		moduleConfig,
		runtime.DefaultServiceBindings(),
		codec.DefaultProviders,
		depinject.Provide(
			ProvideRootStoreConfig,

			basedepinject.ProvideAccount,
			// TODO: do we need multisigdepinject and lockupdepinject?

			basedepinject.ProvideSecp256K1PubKey,
		),
		depinject.Invoke(
			std.RegisterInterfaces,
			std.RegisterLegacyAminoCodec,
		),
	)
}

func NewGCApp(config depinject.Config, outputs ...any) (*GCApp, error) {
	appConfig := depinject.Configs(
		AppConfig(),
		config,
		depinject.Supply(), // Are these necessary if empty?
		depinject.Provide(),
	)

	var app GCApp
	var logger clog.Logger
	var storeBuilder root.Builder
	var appBuilder *runtime.AppBuilder[transaction.Tx]

	outputs = append(
		slices.Clone(outputs),
		&logger,
		&storeBuilder,
		&appBuilder,
		// These were all fields on a new instance of SimApp.
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.UpgradeKeeper,
		&app.StakingKeeper,
	)

	var err error
	if err = depinject.Inject(appConfig, outputs...); err != nil {
		return nil, fmt.Errorf("failed to dep inject: %w", err)
	}

	app.App, err = appBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build app: %w", err)
	}

	app.store = storeBuilder.Get()
	if app.store == nil {
		return nil, fmt.Errorf("store builder returned nil")
	}

	// TODO: simapp does this. Does it matter for gcosmos?
	// app.RegisterUpgradeHandlers()

	// From embedded runtime.App.
	if err = app.LoadLatest(); err != nil {
		return nil, fmt.Errorf("failed to load latest: %w", err)
	}
	return &app, nil
}

func (a *GCApp) Store() store.RootStore {
	return a.store
}

// Close matches the simapp Close.
func (a *GCApp) Close() error {
	if err := a.store.Close(); err != nil {
		return err
	}

	return a.App.Close()
}

// ProvideRootStoreConfig was copied from simapp's ProvideRootStoreConfig.
// depinject requires this to be exported.
func ProvideRootStoreConfig(config runtime.GlobalConfig) (*root.Config, error) {
	return serverstore.UnmarshalConfig(config)
}
