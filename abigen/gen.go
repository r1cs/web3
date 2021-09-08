package abigen

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/laizy/web3/abi"
	"github.com/laizy/web3/compiler"
)

type Config struct {
	Package string
	Output  string
	Name    string
}

func cleanName(str string) string {
	return handleSnakeCase(strings.Trim(str, "_"))
}

func outputArg(str string) string {
	if str == "" {

	}
	return str
}

func handleSnakeCase(str string) string {
	if !strings.Contains(str, "_") {
		return str
	}

	spl := strings.Split(str, "_")
	res := ""
	for indx, elem := range spl {
		if indx != 0 {
			elem = strings.Title(elem)
		}
		res += elem
	}
	return res
}

func funcName(str string) string {
	return strings.Title(handleSnakeCase(str))
}

type tempType struct {
	Name string
	Type string
}

// tmplField is a wrapper around a struct field with binding language
// struct type definition and relative filed name.
type tempStruct struct {
	Name   string
	GoType []*tempType
}

func (a *tempStruct) deepEqual(b *tempStruct) bool {
	if a.Name != b.Name {
		return false
	}
	if len(a.GoType) != len(b.GoType) {
		return false
	}
	for i, _ := range a.GoType {
		if a.GoType[i].Type != b.GoType[i].Type || a.GoType[i].Name != b.GoType[i].Name {
			return false
		}
	}
	return true
}

func encode(typ *abi.Type, structs map[string]*tempStruct) string {
	switch typ.Kind() {
	case abi.KindTuple:

		name := typ.RawName()
		s := &tempStruct{Name: name}
		for _, ty := range typ.TupleElems() {
			goType := encode(ty.Elem, structs)
			name := ty.Name
			s.GoType = append(s.GoType, &tempType{Name: name, Type: goType})
		}

		if name == "" { //input will have no rawName, just use Struct%d now.
			return ""
		}
		if old, exist := structs[name]; exist { //check if two struct have same name but different struct, panic.
			if !s.deepEqual(old) {
				panic(fmt.Sprintf("deprecated struct: %s, should change pkg to different file.", name))
			}
		}
		structs[name] = s
		return name
	case abi.KindAddress:
		return "web3.Address"

	case abi.KindString:
		return "string"

	case abi.KindBool:
		return "bool"

	case abi.KindInt:
		return typ.GoType().String()

	case abi.KindUInt:
		return typ.GoType().String()

	case abi.KindFixedBytes:
		return fmt.Sprintf("[%d]byte", typ.Size())

	case abi.KindBytes:
		return "[]byte"

	case abi.KindSlice:
		return "[]" + encodeSimpleArg(typ.Elem())

	default:
		return fmt.Sprintf("input not done for type: %s", typ.String())
	}
}

func encodeSimpleArg(typ *abi.Type) string {
	switch typ.Kind() {
	case abi.KindAddress:
		return "web3.Address"

	case abi.KindString:
		return "string"

	case abi.KindBool:
		return "bool"

	case abi.KindInt:
		return typ.GoType().String()

	case abi.KindUInt:
		return typ.GoType().String()

	case abi.KindFixedBytes:
		return fmt.Sprintf("[%d]byte", typ.Size())

	case abi.KindBytes:
		return "[]byte"

	case abi.KindSlice:
		return "[]" + encodeSimpleArg(typ.Elem())

	case abi.KindTuple:
		return typ.RawName()
	default:
		return fmt.Sprintf("input not done for type: %s", typ.String())
	}
}

func encodeArg(str interface{}) string {
	arg, ok := str.(*abi.TupleElem)
	if !ok {
		panic("bad 1")
	}
	return encodeSimpleArg(arg.Elem)
}

func tupleLen(tuple interface{}) interface{} {
	if isNil(tuple) {
		return 0
	}
	arg, ok := tuple.(*abi.Type)
	if !ok {
		panic("bad tuple")
	}
	return len(arg.TupleElems())
}

func tupleElems(tuple interface{}) (res []interface{}) {
	if isNil(tuple) {
		return
	}

	arg, ok := tuple.(*abi.Type)
	if !ok {
		panic("bad tuple")
	}
	for _, i := range arg.TupleElems() {
		res = append(res, i)
	}
	return
}

func isNil(c interface{}) bool {
	return c == nil || (reflect.ValueOf(c).Kind() == reflect.Ptr && reflect.ValueOf(c).IsNil())
}

// hasStruct returns an indicator whether the given type is struct, struct slice
// or struct array.
func hasStruct(t *abi.Type) bool {
	switch t.Kind() {
	case abi.KindSlice:
		return hasStruct(t.Elem())
	case abi.KindArray:
		return hasStruct(t.Elem())
	case abi.KindTuple:
		return true
	default:
		return false
	}
}

//readStructFromJson read struct from json cache
func readStructFromJson(fileName string, structs map[string]*tempStruct) error {
	data, err := ioutil.ReadFile(fileName)
	if err == nil {
		structsMem := new(StructsMem)
		if err := structsMem.Read(data); err != nil {
			return err
		}
		structsMem.ToTemp(structs) //set disk cache to memory
	}

	return nil
}
//readStructFromAbi read all struct in abi to structs.
func readStructFromAbi(abi *abi.ABI, structs map[string]*tempStruct) {

	if abi.Constructor != nil && hasStruct(abi.Constructor.Inputs) {
		encode(abi.Constructor.Inputs, structs)
	}
	for _, method := range abi.Methods {
		if hasStruct(method.Inputs) {
			encode(method.Inputs, structs)
		}
	}
	for _, event := range abi.Events {
		if hasStruct(event.Inputs) {
			encode(event.Inputs, structs)
		}
	}

}
func genStruct(abisStr []string, config *Config) error {
	structs := make(map[string]*tempStruct)
	if err := readStructFromJson(filepath.Join(config.Output, "structs.json"), structs); err != nil {
		return fmt.Errorf("read struct from json: %w", err)
	}

	tempStruct, err := template.New("eth-structs").Funcs(map[string]interface{}{"title": strings.Title}).Parse(templateStructStr)
	if err != nil {
		return err
	}
	for _, abiStr := range abisStr {
		// parse abi
		abi, err := abi.NewABI(abiStr)
		if err != nil {
			return err
		}
		readStructFromAbi(abi,structs)
	}

	input := map[string]interface{}{
		"Config":  config,
		"Structs": structs,
	}
	var b bytes.Buffer
	if err := tempStruct.Execute(&b, input); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(config.Output, config.Name+"_structs.go"), b.Bytes(), 0644); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(config.Output, "structs.json"), NewRStructsMem(structs).Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func GenCode(artifacts map[string]*compiler.Artifact, config *Config) error {
	var abisStr []string
	for _, arti := range artifacts {
		if arti.Abi != "" {
			abisStr = append(abisStr, arti.Abi)
		}
	}

	if err := genStruct(abisStr, config); err != nil {
		return fmt.Errorf("genStruct: %s", err)
	}
	funcMap := template.FuncMap{
		"title":      strings.Title,
		"clean":      cleanName,
		"arg":        encodeArg,
		"outputArg":  outputArg,
		"funcName":   funcName,
		"tupleElems": tupleElems,
		"tupleLen":   tupleLen,
	}
	tmplAbi, err := template.New("eth-abi").Funcs(funcMap).Parse(templateAbiStr)
	if err != nil {
		return err
	}
	tmplBin, err := template.New("eth-abi").Funcs(funcMap).Parse(templateBinStr)
	if err != nil {
		return err
	}

	for name, artifact := range artifacts {
		// parse abi
		abi, err := abi.NewABI(artifact.Abi)
		if err != nil {
			return err
		}

		input := map[string]interface{}{
			"Ptr":      "_a",
			"Config":   config,
			"Contract": artifact,
			"Abi":      abi,
			"Name":     name,
		}

		filename := strings.ToLower(name)

		var b bytes.Buffer
		if err := tmplAbi.Execute(&b, input); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(config.Output, filename+".go"), []byte(b.Bytes()), 0644); err != nil {
			return err
		}

		b.Reset()
		if err := tmplBin.Execute(&b, input); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(config.Output, filename+"_artifacts.go"), []byte(b.Bytes()), 0644); err != nil {
			return err
		}
		b.Reset()

	}

	return nil
}

var templateAbiStr = `package {{.Config.Package}}

import (
	"fmt"
	"math/big"

	"github.com/laizy/web3"
	"github.com/laizy/web3/contract"
	"github.com/laizy/web3/jsonrpc"
)

var (
	_ = big.NewInt
	_ = fmt.Printf
)



// {{.Name}} is a solidity contract
type {{.Name}} struct {
	c *contract.Contract
}
{{if .Contract.Bin}}
// Deploy{{.Name}} deploys a new {{.Name}} contract
func Deploy{{.Name}}(provider *jsonrpc.Client, from web3.Address, args ...interface{}) *contract.Txn {
	return contract.DeployContract(provider, from, abi{{.Name}}, bin{{.Name}}, args...)
}
{{end}}
// New{{.Name}} creates a new instance of the contract at a specific address
func New{{.Name}}(addr web3.Address, provider *jsonrpc.Client) *{{.Name}} {
	return &{{.Name}}{c: contract.NewContract(addr, abi{{.Name}}, provider)}
}

// Contract returns the contract object
func ({{.Ptr}} *{{.Name}}) Contract() *contract.Contract {
	return {{.Ptr}}.c
}

// calls
{{range $key, $value := .Abi.Methods}}{{if .Const}}
// {{funcName $key}} calls the {{$key}} method in the solidity contract
func ({{$.Ptr}} *{{$.Name}}) {{funcName $key}}({{range $index, $val := tupleElems .Inputs}}{{if .Name}}{{clean .Name}}{{else}}val{{$index}}{{end}} {{arg .}}, {{end}}block ...web3.BlockNumber) ({{range $index, $val := tupleElems .Outputs}}retval{{$index}} {{arg .}}, {{end}}err error) {
	var out map[string]interface{}
	_ = out // avoid not used compiler error

	{{ $length := tupleLen .Outputs }}{{ if ne $length 0 }}var ok bool{{ end }}

	out, err = {{$.Ptr}}.c.Call("{{$key}}", web3.EncodeBlock(block...){{range $index, $val := tupleElems .Inputs}}, {{if .Name}}{{clean .Name}}{{else}}val{{$index}}{{end}}{{end}})
	if err != nil {
		return
	}

	// decode outputs
	{{range $index, $val := tupleElems .Outputs}}retval{{$index}}, ok = out["{{if .Name}}{{.Name}}{{else}}{{$index}}{{end}}"].({{arg .}})
	if !ok {
		err = fmt.Errorf("failed to encode output at index {{$index}}")
		return
	}
{{end}}
	return
}
{{end}}{{end}}
// txns
{{range $key, $value := .Abi.Methods}}{{if not .Const}}
// {{funcName $key}} sends a {{$key}} transaction in the solidity contract
func ({{$.Ptr}} *{{$.Name}}) {{funcName $key}}({{range $index, $input := tupleElems .Inputs}}{{if $index}}, {{end}}{{clean .Name}} {{arg .}}{{end}}) *contract.Txn {
	return {{$.Ptr}}.c.Txn("{{$key}}"{{range $index, $elem := tupleElems .Inputs}}, {{clean $elem.Name}}{{end}})
}


{{end}}{{end}}`

var templateBinStr = `package {{.Config.Package}}

import (
	"encoding/hex"
	"fmt"

	"github.com/laizy/web3/abi"
)

var abi{{.Name}} *abi.ABI

// {{.Name}}Abi returns the abi of the {{.Name}} contract
func {{.Name}}Abi() *abi.ABI {
	return abi{{.Name}}
}

var bin{{.Name}} []byte
{{if .Contract.Bin}}
// {{.Name}}Bin returns the bin of the {{.Name}} contract
func {{.Name}}Bin() []byte {
	return bin{{.Name}}
}
{{end}}
var binRuntime{{.Name}} []byte
{{if .Contract.BinRuntime}}
// {{.Name}}BinRuntime returns the runtime bin of the {{.Name}} contract
func {{.Name}}BinRuntime() []byte {
	return binRuntime{{.Name}}
}
{{end}}
func init() {
	var err error
	abi{{.Name}}, err = abi.NewABI(abi{{.Name}}Str)
	if err != nil {
		panic(fmt.Errorf("cannot parse {{.Name}} abi: %v", err))
	}
	if len(bin{{.Name}}Str) != 0 {
		bin{{.Name}}, err = hex.DecodeString(bin{{.Name}}Str[2:])
		if err != nil {
			panic(fmt.Errorf("cannot parse {{.Name}} bin: %v", err))
		}
	}
	if len(binRuntime{{.Name}}Str) != 0 {
		binRuntime{{.Name}}, err = hex.DecodeString(binRuntime{{.Name}}Str[2:])
		if err != nil {
			panic(fmt.Errorf("cannot parse {{.Name}} bin runtime: %v", err))
		}
	}
}

var bin{{.Name}}Str = "{{.Contract.Bin}}"

var binRuntime{{.Name}}Str = "{{.Contract.BinRuntime}}"

var abi{{.Name}}Str = ` + "`" + `{{.Contract.Abi}}` + "`\n"
