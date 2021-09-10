package abigen

import (
	"fmt"
	"github.com/laizy/web3/abi"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"text/template"
)

/*
contract TestTuple {

     struct Transaction {
        uint256 timestamp;
        uint256 blockNumber;
        QueueOrigin l1QueueOrigin;
        address l1TxOrigin;
        address entrypoint;
        uint256 gasLimit;
        bytes data;
    }

    enum QueueOrigin {
        SEQUENCER_QUEUE,
        L1TOL2_QUEUE
    }

    function T( Transaction memory a,bytes memory b) public returns (bytes memory){
        return  b;
    }
}
*/

var structTestAbi1 = `
[
	{
		"inputs": [
			{
				"components": [
					{
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "blockNumber",
						"type": "uint256"
					},
					{
						"internalType": "enum TestTuple.QueueOrigin",
						"name": "l1QueueOrigin",
						"type": "uint8"
					},
					{
						"internalType": "address",
						"name": "l1TxOrigin",
						"type": "address"
					},
					{
						"internalType": "address",
						"name": "entrypoint",
						"type": "address"
					},
					{
						"internalType": "uint256",
						"name": "gasLimit",
						"type": "uint256"
					},
					{
						"internalType": "bytes",
						"name": "data",
						"type": "bytes"
					}
				],
				"internalType": "struct TestTuple.Transaction",
				"name": "a",
				"type": "tuple"
			},
			{
				"internalType": "bytes",
				"name": "b",
				"type": "bytes"
			}
		],
		"name": "T",
		"outputs": [
			{
				"internalType": "bytes",
				"name": "",
				"type": "bytes"
			}
		],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]
`

func TestTupleStructs(t *testing.T) {
	assert := require.New(t)
	// parse abi
	abi, err := abi.NewABI(structTestAbi1)
	assert.Nil(err)

	structs := make(map[string]*tempStruct)

	if abi.Constructor != nil && hasStruct(abi.Constructor.Inputs) {
		encode(abi.Constructor.Inputs, structs)
	}
	for _, method := range abi.Methods {
		if hasStruct(method.Inputs) {
			encode(method.Inputs, structs)
		}
	}
	input := map[string]interface{}{
		"Abi":     abi,
		"Structs": structs,
	}
	tmplAbi, err := template.New("test").Funcs(newFuncMap()).Parse(tempS)
	assert.Nil(err)

	assert.Nil(tmplAbi.Execute(os.Stdout, input))
}

var tempS = `
{{$structs := .Structs}}
{{range $structs}}
type {{.Name}} struct {
{{range .GoType}}
{{title .Name}}   {{.Type}} {{end}}
}
{{end}}
`

func injectStructToStructs(ts *tempStruct, structs map[string]*tempStruct) {
	structs[ts.Name] = ts
}

func TestGenStruct(t *testing.T) {
	assert := require.New(t)

	var structs = make(map[string]*tempStruct)
	//config := &Config{Name: "testName", Output: os.Stdout.Name(), Package: "test"}

	abi1, err := abi.NewABI(structTestAbi1)
	assert.Nil(err)

	readStructFromAbi(abi1, structs) //read to structs

	old := len(structs)
	readStructFromAbi(abi1, structs) //read duplicated, but the length won't grow.
	assert.Equal(old, len(structs))

	injectStructToStructs(&tempStruct{Name: "FakeStruct"}, structs)
	readStructFromAbi(abi1, structs) //the length should grow 1
	assert.Equal(old+1, len(structs))

	var oldname string
	for name, _ := range structs {
		oldname = name //read an struct from it
		break
	}
	injectStructToStructs(&tempStruct{Name: oldname}, structs) //will recover oldname

	defer func() {
		e := recover()
		assert.Equal(e.(string), fmt.Sprintf("deprecated struct: %s, should change pkg to different file.", oldname))
	}()
	readStructFromAbi(abi1, structs) //old struct have already in structs, but the inner type is not equal, should panic
}

var evnentTestAbi = `
[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "address",
				"name": "previousOwner",
				"type": "address"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "newOwner",
				"type": "address"
			}
		],
		"name": "OwnershipTransferred",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "string",
				"name": "",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "bytes",
				"name": "",
				"type": "bytes"
			}
		],
		"name": "TestEvent",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "string",
				"name": "",
				"type": "string"
			},
			{
				"indexed": true,
				"internalType": "bytes",
				"name": "",
				"type": "bytes"
			}
		],
		"name": "TestIndexed",
		"type": "event"
	}
]
`

//checkInnerName check struct inner args name not empty and not duplicated
func checkEventInner(event *abi.Event) error {
	dup := make(map[string]bool)
	for _, tuple := range event.Inputs.TupleElems() {
		name := tuple.Name
		if name == "" {
			return fmt.Errorf("the struct inner args name can't be empty")
		}
		if dup[name] {
			return fmt.Errorf("the struct inner args name can't be duplicated")
		}
		dup[name]=true
	}
	return nil
}


func TestGenEvents(t *testing.T) {
	assert := require.New(t)

	eventAbi, err := abi.NewABI(IOVMStateCommitmentChainABI)
	assert.Nil(err)

	var events []*abi.Event
	for _, event := range eventAbi.Events {
		event = optimizeEvent(event)
		assert.Nil(checkEventInner(event))
		events = append(events, event)
	}

	tempevent, err := template.New("test-events").Funcs(newFuncMap()).Parse(templateEvents)
	assert.Nil(err)

	input := map[string]interface{}{
		"Config": &Config{Name: "testEvent"},
		"Name":   "TestGenEvents",
		"Events": events,
	}

	assert.Nil(tempevent.Execute(os.Stdout, input))
}

const IOVMStateCommitmentChainABI = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_batchSize\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_prevTotalElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"StateBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"}],\"name\":\"StateBatchDeleted\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"_batch\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"_shouldStartAtElement\",\"type\":\"uint256\"}],\"name\":\"appendStateBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"}],\"name\":\"deleteStateBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastSequencerTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_lastSequencerTimestamp\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalBatches\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalBatches\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalElements\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"}],\"name\":\"insideFraudProofWindow\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"_inside\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_element\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"}],\"internalType\":\"structLib_OVMCodec.ChainInclusionProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"name\":\"verifyStateCommitment\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"_verified\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"
