package abigen

import (
	"bytes"
	"fmt"
	"github.com/laizy/web3/abi"
	"github.com/laizy/web3/compiler"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
)

var templateEvents = `
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

{{$name:=.Name}}
{{range .Events}}
type {{title .Name}}Params struct{ {{range $k,$v :=tupleElems .Inputs}}
{{title $v.Name}}  {{if $v.Indexed}} {{argTopic $v}} {{else}}  {{arg $v}} {{end}} {{end}}
}
// {{$name}}{{.Name}} represents a {{.Name}} event raised by the {{$name}} contract.
type {{$name}}{{.Name}} struct {
{{title .Name}}  *{{title .Name}}Params
Raw web3.Log // Blockchain specific contextual infos
}
{{end}}


`

//optimizeEvent change inner empty name to arg%d.
func optimizeEvent(event *abi.Event) *abi.Event {
	for j, e := range event.Inputs.TupleElems() {
		if e.Name == "" {
			e.Name = fmt.Sprintf("arg%d", j)
		}
	}
	return event
}

func encodeTopicArg(str interface{}) string {
	arg := encodeArg(str)
	if arg == "string" || arg == "[]byte" {
		arg = "common.Hash"
	}
	return arg
}

func genEvents(artifacts map[string]*compiler.Artifact, config *Config) error {

	funcMap := template.FuncMap{
		"title":      strings.Title,
		"arg":        encodeArg,
		"argTopic":   encodeTopicArg,
		"tupleElems": tupleElems,
	}
	tempevent, err := template.New("eth-events").Funcs(funcMap).Parse(templateEvents)
	if err != nil {
		return err
	}

	var events []*abi.Event
	for name, artifact := range artifacts {
		// parse abi
		abi, err := abi.NewABI(artifact.Abi)
		if err != nil {
			return err
		}
		for _, event := range abi.Events {
			events = append(events, optimizeEvent(event))
		}
		input := map[string]interface{}{
			"Config": config,
			"Name":   name,
			"Events": events,
		}
		var buff bytes.Buffer
		if err := tempevent.Execute(&buff, input); err != nil {
			return err
		}
		filename := strings.ToLower(name)
		if err := ioutil.WriteFile(filepath.Join(config.Output, filename+"_events.go"), buff.Bytes(), 0644); err != nil {
			return err
		}

	}
	return nil
}
