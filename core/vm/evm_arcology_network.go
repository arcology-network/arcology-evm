package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// KernelAPI provides system level function calls supported by arcology platform.
type ArcologyAPIRouterInterface interface {
	Call(caller, callee common.Address, input []byte, origin common.Address, nonce uint64, blockhash common.Hash) (bool, []byte, bool)
}

type ArcologyNetwork struct {
	evm            *EVM
	context        *ScopeContext
	concurrentAPIs ArcologyAPIRouterInterface // Arcology API entrance
}

func NewArcologyNetwork(evm *EVM) *ArcologyNetwork {
	return &ArcologyNetwork{
		evm:     evm,
		context: nil, // only available at run time
	}
}

// Redirect to Arcology API intead
func (this ArcologyNetwork) Redirect(caller ContractRef, addr common.Address, input []byte, gas uint64) (called bool, ret []byte, leftOverGas uint64, err error) {
	if called, ret, ok := this.concurrentAPIs.Call(
		caller.Address(),
		addr,
		input,
		this.evm.Origin,
		this.evm.StateDB.GetNonce(this.evm.Origin),
		this.evm.Context.GetHash(new(big.Int).Sub(this.evm.Context.BlockNumber, big1).Uint64()),
	); called {
		if !ok {
			return true, ret, gas, ErrExecutionReverted
		}
		return true, ret, gas, nil
	}
	return false, ret, gas, nil
}

func (this *ArcologyNetwork) CopyContext(context interface{})             { this.context = context.(*ScopeContext) }
func (this *ArcologyNetwork) Depth() int                                  { return this.evm.depth }
func (this *ArcologyNetwork) CallGasTemp() uint64                         { return this.evm.callGasTemp }
func (this *ArcologyNetwork) GetTxContext() TxContext                     { return this.evm.TxContext }
func (this *ArcologyNetwork) SetApiRouter(api ArcologyAPIRouterInterface) { this.concurrentAPIs = api }
