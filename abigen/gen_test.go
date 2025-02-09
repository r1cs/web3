package abigen

import (
	"go/format"
	"testing"

	"github.com/laizy/web3/abi"
	"github.com/laizy/web3/compiler"
	"github.com/laizy/web3/testutil"
	"github.com/laizy/web3/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var Artifact = func() *compiler.Artifact {
	if testutil.IsSolcInstalled() == false {
		return nil
	}
	code := `
pragma experimental ABIEncoderV2;
contract Sample {
    event Deposit (
        address  indexed _from, // test name with _ will translate to From
        address  indexed _to,
        uint256 _amount,
        bytes _data
    );
	
event NoName(
address indexed,
address
);


	event Transfer (
		address indexed from,
		address indexed to,
		address indexed amount
	);

    struct Transaction {
        uint256 timestamp;
        QueueOrigin l1QueueOrigin;
        address entrypoint;
        bytes data;
    }

	struct Output {
		uint256 num;
		bytes data;
		address l1QueueOriginAddress;
	}

    enum QueueOrigin {
        SEQUENCER_QUEUE,
        L1TOL2_QUEUE
    }

	 constructor(){ emit Deposit(msg.sender,msg.sender,100000,bytes("test")); emit NoName(msg.sender,msg.sender);}

    function TestStruct(Transaction memory a,bytes memory b) public returns (bytes memory){
        return  b;
    }

	function TestOutPut() public view returns (Output memory){
		Output memory haha;
		return haha;
	}

    function getTxes(Transaction[] memory txes) external view returns (Transaction[] memory) {
        return txes;
    }
}
`
	solc := &compiler.Solidity{Path: "solc"}
	output, err := solc.CompileCode(code)
	utils.Ensure(err)
	return output["<stdin>:Sample"]
}()

func TestCodeGen(t *testing.T) {
	if testutil.IsSolcInstalled() == false {
		t.Skipf("skipping since solidity is not installed")
	}
	config := &Config{
		Package: "binding",
		Output:  "sample",
		Name:    "Sample",
	}

	artifacts := map[string]*compiler.Artifact{
		"Sample": Artifact,
	}
	res, err := NewGenerator(config, artifacts).Gen()
	assert.Nil(t, err)

	expected, _ := format.Source([]byte(`package binding

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/laizy/web3"
	"github.com/laizy/web3/contract"
	"github.com/laizy/web3/crypto"
	"github.com/laizy/web3/jsonrpc"
	"github.com/laizy/web3/utils"
	"github.com/mitchellh/mapstructure"
)

var (
	_ = json.Unmarshal
	_ = big.NewInt
	_ = fmt.Printf
	_ = utils.JsonStr
	_ = mapstructure.Decode
	_ = crypto.Keccak256Hash
)

// Sample is a solidity contract
type Sample struct {
	c *contract.Contract
}

// DeploySample deploys a new Sample contract
func DeploySample(provider *jsonrpc.Client, from web3.Address, args ...interface{}) *contract.Txn {
	return contract.DeployContract(provider, from, abiSample, binSample, args...)
}

// NewSample creates a new instance of the contract at a specific address
func NewSample(addr web3.Address, provider *jsonrpc.Client) *Sample {
	return &Sample{c: contract.NewContract(addr, abiSample, provider)}
}

// Contract returns the contract object
func (_a *Sample) Contract() *contract.Contract {
	return _a.c
}

// calls

// TestOutPut calls the TestOutPut method in the solidity contract
func (_a *Sample) TestOutPut(block ...web3.BlockNumber) (retval0 Output, err error) {
	var out map[string]interface{}
	_ = out // avoid not used compiler error

	out, err = _a.c.Call("TestOutPut", web3.EncodeBlock(block...))
	if err != nil {
		return
	}

	// decode outputs

	if err = mapstructure.Decode(out["0"], &retval0); err != nil {
		err = fmt.Errorf("failed to encode output at index 0")
	}

	return
}

// GetTxes calls the getTxes method in the solidity contract
func (_a *Sample) GetTxes(txes []Transaction, block ...web3.BlockNumber) (retval0 []Transaction, err error) {
	var out map[string]interface{}
	_ = out // avoid not used compiler error

	out, err = _a.c.Call("getTxes", web3.EncodeBlock(block...), txes)
	if err != nil {
		return
	}

	// decode outputs

	if err = mapstructure.Decode(out["0"], &retval0); err != nil {
		err = fmt.Errorf("failed to encode output at index 0")
	}

	return
}

// txns

// TestStruct sends a TestStruct transaction in the solidity contract
func (_a *Sample) TestStruct(a Transaction, b []byte) *contract.Txn {
	return _a.c.Txn("TestStruct", a, b)
}

// events

var DepositEventID = crypto.Keccak256Hash([]byte("Deposit(address,address,uint256,bytes)"))

func (_a *Sample) DepositTopicFilter(from []web3.Address, to []web3.Address) [][]web3.Hash {

	var fromRule []interface{}
	for _, _fromItem := range from {
		fromRule = append(fromRule, _fromItem)
	}

	var toRule []interface{}
	for _, _toItem := range to {
		toRule = append(toRule, _toItem)
	}

	var query [][]interface{}
	query = append(query, []interface{}{DepositEventID}, fromRule, toRule)

	topics, err := contract.MakeTopics(query...)
	utils.Ensure(err)

	return topics
}

func (_a *Sample) FilterDepositEvent(from []web3.Address, to []web3.Address, startBlock uint64, endBlock ...uint64) ([]*DepositEvent, error) {
	topic := _a.DepositTopicFilter(from, to)

	logs, err := _a.c.FilterLogsWithTopic(topic, startBlock, endBlock...)
	if err != nil {
		return nil, err
	}
	res := make([]*DepositEvent, 0)
	evts := _a.c.Abi.Events["Deposit"]
	for _, log := range logs {
		args, err := evts.ParseLog(log)
		if err != nil {
			return nil, err
		}
		var evtItem DepositEvent
		err = json.Unmarshal([]byte(utils.JsonStr(args)), &evtItem)
		if err != nil {
			return nil, err
		}
		evtItem.Raw = log
		res = append(res, &evtItem)
	}
	return res, nil
}

var NoNameEventID = crypto.Keccak256Hash([]byte("NoName(address,address)"))

func (_a *Sample) NoNameTopicFilter(arg0 []web3.Address) [][]web3.Hash {

	var arg0Rule []interface{}
	for _, arg0Item := range arg0 {
		arg0Rule = append(arg0Rule, arg0Item)
	}

	var query [][]interface{}
	query = append(query, []interface{}{NoNameEventID}, arg0Rule)

	topics, err := contract.MakeTopics(query...)
	utils.Ensure(err)

	return topics
}

func (_a *Sample) FilterNoNameEvent(arg0 []web3.Address, startBlock uint64, endBlock ...uint64) ([]*NoNameEvent, error) {
	topic := _a.NoNameTopicFilter(arg0)

	logs, err := _a.c.FilterLogsWithTopic(topic, startBlock, endBlock...)
	if err != nil {
		return nil, err
	}
	res := make([]*NoNameEvent, 0)
	evts := _a.c.Abi.Events["NoName"]
	for _, log := range logs {
		args, err := evts.ParseLog(log)
		if err != nil {
			return nil, err
		}
		var evtItem NoNameEvent
		err = json.Unmarshal([]byte(utils.JsonStr(args)), &evtItem)
		if err != nil {
			return nil, err
		}
		evtItem.Raw = log
		res = append(res, &evtItem)
	}
	return res, nil
}

var TransferEventID = crypto.Keccak256Hash([]byte("Transfer(address,address,address)"))

func (_a *Sample) TransferTopicFilter(from []web3.Address, to []web3.Address, amount []web3.Address) [][]web3.Hash {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	var query [][]interface{}
	query = append(query, []interface{}{TransferEventID}, fromRule, toRule, amountRule)

	topics, err := contract.MakeTopics(query...)
	utils.Ensure(err)

	return topics
}

func (_a *Sample) FilterTransferEvent(from []web3.Address, to []web3.Address, amount []web3.Address, startBlock uint64, endBlock ...uint64) ([]*TransferEvent, error) {
	topic := _a.TransferTopicFilter(from, to, amount)

	logs, err := _a.c.FilterLogsWithTopic(topic, startBlock, endBlock...)
	if err != nil {
		return nil, err
	}
	res := make([]*TransferEvent, 0)
	evts := _a.c.Abi.Events["Transfer"]
	for _, log := range logs {
		args, err := evts.ParseLog(log)
		if err != nil {
			return nil, err
		}
		var evtItem TransferEvent
		err = json.Unmarshal([]byte(utils.JsonStr(args)), &evtItem)
		if err != nil {
			return nil, err
		}
		evtItem.Raw = log
		res = append(res, &evtItem)
	}
	return res, nil
}
`))

	assert.Equal(t, string(expected), string(res.AbiFiles[0].Code))
}

func TestTupleStructs(t *testing.T) {
	if testutil.IsSolcInstalled() == false {
		t.Skipf("skipping since solidity is not installed")
	}
	assert := require.New(t)
	code, err := NewStructDefExtractor().ExtractFromAbi(abi.MustNewABI(Artifact.Abi)).RenderGoCode("binding")
	assert.Nil(err)

	expected, _ := format.Source([]byte(`package binding

import (
	"fmt"
	"math/big"

	"github.com/laizy/web3"
)

var (
	_ = big.NewInt
	_ = fmt.Printf
	_ = web3.HexToAddress
)

type DepositEvent struct {
	From   web3.Address
	To     web3.Address
	Amount *big.Int
	Data   []byte

	Raw *web3.Log
}

type NoNameEvent struct {
	Arg0 web3.Address
	Arg1 web3.Address

	Raw *web3.Log
}

type Output struct {
	Num                  *big.Int
	Data                 []byte
	L1QueueOriginAddress web3.Address
}

type Transaction struct {
	Timestamp     *big.Int
	L1QueueOrigin uint8
	Entrypoint    web3.Address
	Data          []byte
}

type TransferEvent struct {
	From   web3.Address
	To     web3.Address
	Amount web3.Address

	Raw *web3.Log
}
`))

	assert.Equal(string(expected), code)
}

func TestGenStruct(t *testing.T) {
	if testutil.IsSolcInstalled() == false {
		t.Skipf("skipping since solidity is not installed")
	}
	assert := require.New(t)

	defs := NewStructDefExtractor()
	abi1, err := abi.NewABI(Artifact.Abi)
	assert.Nil(err)

	defs.ExtractFromAbi(abi1)

	assert.NotNil(defs.Defs["Output"]) //output struct is generated

	old := len(defs.Defs)
	defs.ExtractFromAbi(abi1)
	assert.Equal(old, len(defs.Defs)) // test dulplicate case

	var oldname string
	for name := range defs.Defs {
		oldname = name //read an struct from it
		break
	}

	defs.Defs[oldname] = &StructDef{Name: oldname, IsEvent: defs.Defs[oldname].IsEvent}
	assert.PanicsWithError(ErrConflictDef.Error(), func() {
		defs.ExtractFromAbi(abi1)
	})
}

func TestEncodeTopic(t *testing.T) {
	arg := &abi.ArgumentStr{
		Type: "tuple",
		Components: []*abi.ArgumentStr{
			{
				Name:    "",
				Indexed: true,
				Type:    "string",
			},
			{
				Name:    "",
				Indexed: true,
				Type:    "bytes",
			},
		},
	}

	assert := require.New(t)
	typ, err := abi.NewTypeFromArgument(arg)
	assert.Nil(err)

	assert.Equal("web3.Hash", encodeTopicArg(typ.TupleElems()[0]))
	assert.Equal("web3.Hash", encodeTopicArg(typ.TupleElems()[1]))

}
