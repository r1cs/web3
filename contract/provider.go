package contract

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/laizy/web3"
	"github.com/laizy/web3/executor"
	"github.com/laizy/web3/jsonrpc"
	"github.com/laizy/web3/utils"
	"github.com/laizy/web3/wallet"
)

type Signer struct {
	*wallet.Key
	signer wallet.Signer
	*jsonrpc.Client
	Executor *executor.Executor
	Submit   bool
	Nonce    uint64 // only used when in simulate mode
}

func NewSigner(hexPrivKey string, client *jsonrpc.Client, chainId uint64) *Signer {
	hexPrivKey = strings.TrimPrefix(hexPrivKey, "0x")
	key, err := hex.DecodeString(hexPrivKey)
	utils.Ensure(err)
	account, err := wallet.NewWalletFromPrivKey(key)
	utils.Ensure(err)

	signer := wallet.NewEIP155Signer(chainId)

	nonce, err := client.Eth().GetNonce(account.Address(), web3.Latest)
	utils.Ensure(err)

	result := &Signer{
		Key:      account,
		signer:   signer,
		Client:   client,
		Executor: executor.NewExecutor(client),
		Nonce:    nonce,
	}

	return result
}

func (self *Signer) SignTx(tx *web3.Transaction) *web3.Transaction {
	txn, err := self.signer.SignTx(tx, self.Key)
	utils.Ensure(err)
	return txn
}

func (self *Signer) SendTransaction(tx *web3.Transaction) *web3.Receipt {
	if self.Submit == false {
		result, receipt := self.ExecuteTxn(tx)
		if result.Err != nil {
			panic(fmt.Errorf("execution reverted: %s", result.RevertReason))
		}
		return receipt
	}

	if len(tx.R) == 0 {
		tx = self.SignTx(tx)
	}
	hs, err := self.Eth().SendRawTransaction(tx.MarshalRLP())
	utils.Ensure(err)
	return self.WaitTx(hs)
}

func (self *Signer) ExecuteTxn(tx *web3.Transaction) (*web3.ExecutionResult, *web3.Receipt) {
	num, err := self.Client.Eth().BlockNumber()
	utils.Ensure(err)
	result, receipt, err := self.Executor.ExecuteTransaction(tx, executor.Eip155Context{
		Height:    num + 1,
		Timestamp: uint64(time.Now().Unix()),
	})

	utils.Ensure(err)
	return result, receipt
}

func (self *Signer) WaitTx(hs web3.Hash) *web3.Receipt {
	for {
		receipt, err := self.Client.Eth().GetTransactionReceipt(hs)
		if err != nil {
			if err.Error() != "not found" {
				panic(err)
			}
		}
		if receipt != nil {
			return receipt
		}
	}
}

func (self *Signer) TransferEther(to web3.Address, value *big.Int, msg string) *web3.Transaction {
	nonce, err := self.Client.Eth().GetNonce(self.Key.Address(), web3.Pending)
	utils.Ensure(err)
	price, err := self.Client.Eth().GasPrice()
	utils.Ensure(err)

	tx := &web3.Transaction{
		To:       &to,
		GasPrice: price,
		Gas:      41000 + uint64(len(msg))*100,
		Value:    value,
		Nonce:    nonce,
		Input:    []byte(msg),
	}

	return self.SignTx(tx)
}

func (e *Signer) GetNonce(blockNumber web3.BlockNumber) (uint64, error) {
	return e.Eth().GetNonce(e.Address(), blockNumber)
}
