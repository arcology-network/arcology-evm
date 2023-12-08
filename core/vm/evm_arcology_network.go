package vm

import (
	"math/big"

	"github.com/arcology-network/common-lib/codec"
	commontypes "github.com/arcology-network/common-lib/common"
	"github.com/ethereum/go-ethereum/common"
)

// KernelAPI provides system level function calls supported by arcology platform.
type ArcologyAPIRouterInterface interface {
	Call(caller, callee [20]byte, input []byte, origin [20]byte, nonce uint64, blockhash common.Hash) (bool, []byte, bool, int64)
}

type ArcologyNetwork struct {
	evm            *EVM
	callerContract ContractRef
	CallContext    *ScopeContext              // only available at run time
	APIs           ArcologyAPIRouterInterface // Arcology API entrance
}

func NewArcologyNetwork(evm *EVM) *ArcologyNetwork {
	return &ArcologyNetwork{
		evm: evm,
		// context: nil, // only available at run time
	}
}

// Redirect to Arcology API intead
func (this ArcologyNetwork) Call(callerContract ContractRef, addr common.Address, input []byte, gas uint64) (called bool, ret []byte, leftOverGas uint64, err error) {
	this.callerContract = callerContract
	if called, ret, ok, gasUsed := this.APIs.Call(
		this.callerContract.Address(),
		addr,
		input,
		this.evm.Origin,
		this.evm.StateDB.GetNonce(this.evm.Origin),
		this.evm.Context.GetHash(new(big.Int).Sub(this.evm.Context.BlockNumber, big1).Uint64()),
	); called {
		if gasUsed < 0 {
			leftOverGas = gas + uint64(gasUsed*-1)
		} else {
			leftOverGas = gas - uint64(gasUsed)
		}

		if !ok {
			return true, ret, leftOverGas, ErrExecutionReverted
		}
		return true, ret, leftOverGas, nil
	}
	return false, ret, gas, nil
}

func (this *ArcologyNetwork) GetCallData() []byte {
	if this.CallContext.Contract != nil {
		return (this.CallContext.Contract.Input)
	}
	return []byte{}
}

func (this *ArcologyNetwork) CopyContext(context interface{}) {
	this.CallContext = context.(*ScopeContext)
}

func (this *ArcologyNetwork) Depth() int { return this.evm.depth }

func (this *ArcologyNetwork) CallHierarchy() [][]byte {
	buffers := [][]byte{
		this.CallContext.Contract.Input[:4],
		codec.Bytes20(this.CallContext.Contract.Address()).Encode(),
	}

	if commontypes.IsType[*Contract](this.CallContext.Contract.caller) { // Not a contract
		caller := this.CallContext.Contract.caller
		for {
			if !commontypes.IsType[*Contract](caller) { // Not a contract
				break
			}
			buffers = append(buffers, caller.(*Contract).Input[:4])
			buffers = append(buffers, codec.Bytes20(caller.Address()).Encode())

			caller = caller.(*Contract).caller
		}
	}
	return buffers
}
