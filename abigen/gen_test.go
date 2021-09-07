package abigen

import (
	"github.com/laizy/web3/abi"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
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

var structTestAbi = `
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
	abi, err := abi.NewABI(structTestAbi)
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
	tmplAbi, err := template.New("test").Funcs(map[string]interface{}{"title": strings.Title}).Parse(tempS)
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
