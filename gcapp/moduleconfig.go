package gcapp

import (
	"time"

	accountsmodulev1 "cosmossdk.io/api/cosmos/accounts/module/v1"
	runtimev2 "cosmossdk.io/api/cosmos/app/runtime/v2"
	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	authzmodulev1 "cosmossdk.io/api/cosmos/authz/module/v1"
	bankmodulev1 "cosmossdk.io/api/cosmos/bank/module/v1"
	circuitmodulev1 "cosmossdk.io/api/cosmos/circuit/module/v1"
	consensusmodulev1 "cosmossdk.io/api/cosmos/consensus/module/v1"
	distrmodulev1 "cosmossdk.io/api/cosmos/distribution/module/v1"
	epochsmodulev1 "cosmossdk.io/api/cosmos/epochs/module/v1"
	evidencemodulev1 "cosmossdk.io/api/cosmos/evidence/module/v1"
	feegrantmodulev1 "cosmossdk.io/api/cosmos/feegrant/module/v1"
	genutilmodulev1 "cosmossdk.io/api/cosmos/genutil/module/v1"
	govmodulev1 "cosmossdk.io/api/cosmos/gov/module/v1"
	groupmodulev1 "cosmossdk.io/api/cosmos/group/module/v1"
	mintmodulev1 "cosmossdk.io/api/cosmos/mint/module/v1"
	nftmodulev1 "cosmossdk.io/api/cosmos/nft/module/v1"
	poolmodulev1 "cosmossdk.io/api/cosmos/protocolpool/module/v1"
	slashingmodulev1 "cosmossdk.io/api/cosmos/slashing/module/v1"
	stakingmodulev1 "cosmossdk.io/api/cosmos/staking/module/v1"
	txconfigv1 "cosmossdk.io/api/cosmos/tx/config/v1"
	upgrademodulev1 "cosmossdk.io/api/cosmos/upgrade/module/v1"
	validatemodulev1 "cosmossdk.io/api/cosmos/validate/module/v1"
	vestingmodulev1 "cosmossdk.io/api/cosmos/vesting/module/v1"
	"cosmossdk.io/depinject/appconfig"
	"cosmossdk.io/x/accounts"
	"cosmossdk.io/x/authz"
	banktypes "cosmossdk.io/x/bank/types"
	bankv2types "cosmossdk.io/x/bank/v2/types"
	bankmodulev2 "cosmossdk.io/x/bank/v2/types/module"
	circuittypes "cosmossdk.io/x/circuit/types"
	consensustypes "cosmossdk.io/x/consensus/types"
	distrtypes "cosmossdk.io/x/distribution/types"
	epochstypes "cosmossdk.io/x/epochs/types"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	govtypes "cosmossdk.io/x/gov/types"
	"cosmossdk.io/x/group"
	minttypes "cosmossdk.io/x/mint/types"
	"cosmossdk.io/x/nft"
	pooltypes "cosmossdk.io/x/protocolpool/types"
	slashingtypes "cosmossdk.io/x/slashing/types"
	stakingtypes "cosmossdk.io/x/staking/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/validate"
	"google.golang.org/protobuf/types/known/durationpb"

	// At least some of these are required for proper gRPC behavior.
	_ "cosmossdk.io/api/cosmos/reflection/v1"
	_ "cosmossdk.io/x/accounts"
	_ "cosmossdk.io/x/accounts/v1"
	_ "cosmossdk.io/x/gov"
	_ "cosmossdk.io/x/upgrade"
)

// Copy of the same variable from simapp/v2/app_config.go.
var moduleAccPerms = []*authmodulev1.ModuleAccountPermission{
	{Account: authtypes.FeeCollectorName},
	{Account: distrtypes.ModuleName},
	{Account: pooltypes.ModuleName},
	{Account: pooltypes.StreamAccount},
	{Account: pooltypes.ProtocolPoolDistrAccount},
	{Account: minttypes.ModuleName, Permissions: []string{authtypes.Minter}},
	{Account: stakingtypes.BondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
	{Account: stakingtypes.NotBondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
	{Account: govtypes.ModuleName, Permissions: []string{authtypes.Burner}},
	{Account: nft.ModuleName},
}

// Copy of blockAccAddrs from simapp/v2/app_config.go.
var blockAccAddrs = []string{
	authtypes.FeeCollectorName,
	distrtypes.ModuleName,
	minttypes.ModuleName,
	stakingtypes.BondedPoolName,
	stakingtypes.NotBondedPoolName,
	nft.ModuleName,
	// We allow the following module accounts to receive funds:
	// govtypes.ModuleName
	// pooltypes.ModuleName
}

// moduleConfig is virtually a copy-paste of simapp's ModuleConfig.
var moduleConfig = appconfig.Compose(&appv1alpha1.Config{
	Modules: []*appv1alpha1.ModuleConfig{
		{
			Name: runtime.ModuleName,
			Config: appconfig.WrapAny(&runtimev2.Module{
				AppName: "gcosmos",
				// NOTE: upgrade module is required to be prioritized
				PreBlockers: []string{
					upgradetypes.ModuleName,
				},
				// During begin block slashing happens after distr.BeginBlocker so that
				// there is nothing left over in the validator fee pool, so as to keep the
				// CanWithdrawInvariant invariant.
				// NOTE: staking module is required if HistoricalEntries param > 0
				BeginBlockers: []string{
					minttypes.ModuleName,
					distrtypes.ModuleName,
					pooltypes.ModuleName,
					slashingtypes.ModuleName,
					evidencetypes.ModuleName,
					stakingtypes.ModuleName,
					authz.ModuleName,
					epochstypes.ModuleName,
				},
				EndBlockers: []string{
					govtypes.ModuleName,
					stakingtypes.ModuleName,
					feegrant.ModuleName,
					group.ModuleName,
					pooltypes.ModuleName,
				},
				OverrideStoreKeys: []*runtimev2.StoreKeyConfig{
					{
						ModuleName: authtypes.ModuleName,
						KvStoreKey: "acc",
					},
					{
						ModuleName: accounts.ModuleName,
						KvStoreKey: accounts.StoreKey,
					},
				},
				// NOTE: The genutils module must occur after staking so that pools are
				// properly initialized with tokens from genesis accounts.
				// NOTE: The genutils module must also occur after auth so that it can access the params from auth.
				InitGenesis: []string{
					consensustypes.ModuleName,
					accounts.ModuleName,
					authtypes.ModuleName,
					banktypes.ModuleName,
					bankv2types.ModuleName,
					distrtypes.ModuleName,
					stakingtypes.ModuleName,
					slashingtypes.ModuleName,
					govtypes.ModuleName,
					minttypes.ModuleName,
					genutiltypes.ModuleName,
					evidencetypes.ModuleName,
					authz.ModuleName,
					feegrant.ModuleName,
					nft.ModuleName,
					group.ModuleName,
					upgradetypes.ModuleName,
					vestingtypes.ModuleName,
					circuittypes.ModuleName,
					pooltypes.ModuleName,
					epochstypes.ModuleName,
				},
				// When ExportGenesis is not specified, the export genesis module order
				// is equal to the init genesis order
				// ExportGenesis: []string{},
				// Uncomment if you want to set a custom migration order here.
				// OrderMigrations: []string{},
				// TODO GasConfig was added to the config in runtimev2.  Where/how was it set in v1?
				GasConfig: &runtimev2.GasConfig{
					ValidateTxGasLimit: 100_000,
					QueryGasLimit:      100_000,
					SimulationGasLimit: 100_000,
				},
				// SkipStoreKeys is an optional list of store keys to skip when constructing the
				// module's keeper. This is useful when a module does not have a store key.
				SkipStoreKeys: []string{
					authtxconfig.DepinjectModuleName,
					validate.ModuleName,
				},
			}),
		},
		{
			Name:   authtxconfig.DepinjectModuleName, // x/auth/tx/config depinject module (not app module), use to provide tx configuration
			Config: appconfig.WrapAny(&txconfigv1.Config{}),
		},
		{
			Name:   validate.ModuleName,
			Config: appconfig.WrapAny(&validatemodulev1.Module{}),
		},
		{
			Name: authtypes.ModuleName,
			Config: appconfig.WrapAny(&authmodulev1.Module{
				Bech32Prefix:             "cosmos",
				ModuleAccountPermissions: moduleAccPerms,
				// By default modules authority is the governance module. This is configurable with the following:
				// Authority: "group", // A custom module authority can be set using a module name
				// Authority: "cosmos1cwwv22j5ca08ggdv9c2uky355k908694z577tv", // or a specific address
			}),
		},
		{
			Name:   vestingtypes.ModuleName,
			Config: appconfig.WrapAny(&vestingmodulev1.Module{}),
		},
		{
			Name: banktypes.ModuleName,
			Config: appconfig.WrapAny(&bankmodulev1.Module{
				BlockedModuleAccountsOverride: blockAccAddrs,
			}),
		},
		{
			Name: stakingtypes.ModuleName,
			Config: appconfig.WrapAny(&stakingmodulev1.Module{
				// NOTE: specifying a prefix is only necessary when using bech32 addresses
				// If not specified, the auth Bech32Prefix appended with "valoper" and "valcons" is used by default
				Bech32PrefixValidator: "cosmosvaloper",
				Bech32PrefixConsensus: "cosmosvalcons",
			}),
		},
		{
			Name:   slashingtypes.ModuleName,
			Config: appconfig.WrapAny(&slashingmodulev1.Module{}),
		},
		{
			Name:   genutiltypes.ModuleName,
			Config: appconfig.WrapAny(&genutilmodulev1.Module{}),
		},
		{
			Name:   authz.ModuleName,
			Config: appconfig.WrapAny(&authzmodulev1.Module{}),
		},
		{
			Name:   upgradetypes.ModuleName,
			Config: appconfig.WrapAny(&upgrademodulev1.Module{}),
		},
		{
			Name:   distrtypes.ModuleName,
			Config: appconfig.WrapAny(&distrmodulev1.Module{}),
		},
		{
			Name:   evidencetypes.ModuleName,
			Config: appconfig.WrapAny(&evidencemodulev1.Module{}),
		},
		{
			Name:   minttypes.ModuleName,
			Config: appconfig.WrapAny(&mintmodulev1.Module{}),
		},
		{
			Name: group.ModuleName,
			Config: appconfig.WrapAny(&groupmodulev1.Module{
				MaxExecutionPeriod: durationpb.New(time.Second * 1209600),
				MaxMetadataLen:     255,
			}),
		},
		{
			Name:   nft.ModuleName,
			Config: appconfig.WrapAny(&nftmodulev1.Module{}),
		},
		{
			Name:   feegrant.ModuleName,
			Config: appconfig.WrapAny(&feegrantmodulev1.Module{}),
		},
		{
			Name:   govtypes.ModuleName,
			Config: appconfig.WrapAny(&govmodulev1.Module{}),
		},
		{
			Name:   consensustypes.ModuleName,
			Config: appconfig.WrapAny(&consensusmodulev1.Module{}),
		},
		{
			Name:   accounts.ModuleName,
			Config: appconfig.WrapAny(&accountsmodulev1.Module{}),
		},
		{
			Name:   circuittypes.ModuleName,
			Config: appconfig.WrapAny(&circuitmodulev1.Module{}),
		},
		{
			Name:   pooltypes.ModuleName,
			Config: appconfig.WrapAny(&poolmodulev1.Module{}),
		},
		{
			Name:   epochstypes.ModuleName,
			Config: appconfig.WrapAny(&epochsmodulev1.Module{}),
		},
		{
			Name:   bankv2types.ModuleName,
			Config: appconfig.WrapAny(&bankmodulev2.Module{}),
		},
	},
})
