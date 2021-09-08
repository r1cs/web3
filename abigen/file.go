package abigen

import "encoding/json"

type StructsMem struct {
	Memories []MemStruct
}

type MemStruct struct {
	Temp *tempStruct
}

func(s *StructsMem)Bytes()([]byte){
	b,err:= json.MarshalIndent(s,""," ")
	if err!= nil {
		panic(err)
	}
	return b
}

func(s *StructsMem)Read(data []byte)error{
	return json.Unmarshal(data,s)
}

func (s *StructsMem) ToTemp(structs map[string]*tempStruct) {
	for _, m := range s.Memories {
		structs[m.Temp.Name] = m.Temp
	}
}

func NewRStructsMem(structs map[string]*tempStruct) *StructsMem {
	if len(structs) == 0 {
		return nil
	}
	m := new(StructsMem)
	for _, s := range structs {
		m.Memories = append(m.Memories, MemStruct{ Temp: s})
	}
	return m
}

var templateStructStr = `
package {{.Config.Package}}

import (
	"fmt"
	"math/big"

	"github.com/laizy/web3"

)

var (
	_ = big.NewInt
	_ = fmt.Printf
)

{{$structs := .Structs}}
{{range $structs}}
type {{.Name}} struct {
{{range .GoType}}
{{title .Name}}   {{.Type}} {{end}}
}

{{end}}
`
