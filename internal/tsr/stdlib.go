package tsr

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

func (rt *Runtime) registerStdlib() {
	builtins := map[string]func(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error){
		"say":            builtinSay(rt),
		"type":           builtinType,
		"len":            builtinLen,
		"to_string":      builtinToString,
		"to_number":      builtinToNumber,
		"now_ms":         builtinNowMs,
		"sleep_ms":       builtinSleepMs,
		"json_parse":     builtinJsonParse,
		"json_stringify": builtinJsonStringify,
		"read_file":      rt.builtinReadFile(),
		"write_file":     rt.builtinWriteFile(),
		"append_file":    rt.builtinAppendFile(),
		"http_get":       rt.builtinHttpGet(),
		"http_post":      rt.builtinHttpPost(),
	}
	for name, fn := range builtins {
		localFn := fn
		localName := name
		rt.global.set(localName, &Value{
			Type: ValFunc,
			FuncVal: &Function{
				Name:    localName,
				Builtin: localFn,
			},
		})
	}
}

func builtinSay(rt *Runtime) func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
	return func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("say() expects 1 argument")
		}
		fmt.Fprintln(rt.stdout, valueToString(args[0]))
		return nilValue, nil
	}
}

func builtinType(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("type() expects 1 argument")
	}
	return strVal(valueTypeName(args[0])), nil
}

func builtinLen(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len() expects 1 argument")
	}
	switch args[0].Type {
	case ValString:
		return numVal(new(big.Rat).SetInt64(int64(len([]rune(args[0].StrVal))))), nil
	case ValList:
		return numVal(new(big.Rat).SetInt64(int64(len(args[0].ListVal)))), nil
	case ValDict:
		return numVal(new(big.Rat).SetInt64(int64(len(args[0].DictVal)))), nil
	default:
		return nil, fmt.Errorf("len() expects string, list, or dict")
	}
}

func builtinToString(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("to_string() expects 1 argument")
	}
	return strVal(valueToString(args[0])), nil
}

func builtinToNumber(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("to_number() expects 1 argument")
	}
	switch args[0].Type {
	case ValNumber:
		return args[0], nil
	case ValString:
		r := new(big.Rat)
		_, ok := r.SetString(args[0].StrVal)
		if !ok {
			return nil, fmt.Errorf("to_number(): cannot convert %q to number", args[0].StrVal)
		}
		return numVal(r), nil
	case ValBool:
		if args[0].BoolVal {
			return numVal(new(big.Rat).SetInt64(1)), nil
		}
		return numVal(new(big.Rat).SetInt64(0)), nil
	default:
		return nil, fmt.Errorf("to_number(): cannot convert %s", valueTypeName(args[0]))
	}
}

func builtinNowMs(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error) {
	ms := time.Now().UnixMilli()
	return numVal(new(big.Rat).SetInt64(ms)), nil
}

func builtinSleepMs(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sleep_ms() expects 1 argument")
	}
	if args[0].Type != ValNumber {
		return nil, fmt.Errorf("sleep_ms() expects number")
	}
	f, _ := args[0].NumVal.Float64()
	time.Sleep(time.Duration(f) * time.Millisecond)
	return nilValue, nil
}

func builtinJsonParse(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("json_parse() expects 1 argument")
	}
	if args[0].Type != ValString {
		return nil, fmt.Errorf("json_parse() expects string")
	}
	var raw interface{}
	if err := json.Unmarshal([]byte(args[0].StrVal), &raw); err != nil {
		return nil, fmt.Errorf("json_parse(): %s", err)
	}
	return jsonToValue(raw)
}

func jsonToValue(raw interface{}) (*Value, error) {
	switch v := raw.(type) {
	case nil:
		return nilValue, nil
	case bool:
		return boolVal(v), nil
	case float64:
		r := new(big.Rat).SetFloat64(v)
		return numVal(r), nil
	case string:
		return strVal(v), nil
	case []interface{}:
		var list []*Value
		for _, elem := range v {
			val, err := jsonToValue(elem)
			if err != nil {
				return nil, err
			}
			list = append(list, val)
		}
		return &Value{Type: ValList, ListVal: list}, nil
	case map[string]interface{}:
		d := make(map[string]*Value)
		for k, val := range v {
			tsrVal, err := jsonToValue(val)
			if err != nil {
				return nil, err
			}
			d[k] = tsrVal
		}
		return &Value{Type: ValDict, DictVal: d}, nil
	default:
		return nilValue, nil
	}
}

func builtinJsonStringify(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("json_stringify() expects 1 argument")
	}
	raw := valueToJson(args[0])
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("json_stringify(): %s", err)
	}
	return strVal(string(b)), nil
}

func valueToJson(v *Value) interface{} {
	switch v.Type {
	case ValNil:
		return nil
	case ValBool:
		return v.BoolVal
	case ValNumber:
		f, _ := v.NumVal.Float64()
		return f
	case ValString:
		return v.StrVal
	case ValList:
		var list []interface{}
		for _, e := range v.ListVal {
			list = append(list, valueToJson(e))
		}
		return list
	case ValDict:
		d := make(map[string]interface{})
		for k, val := range v.DictVal {
			d[k] = valueToJson(val)
		}
		return d
	default:
		return nil
	}
}

func (rt *Runtime) builtinReadFile() func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
	return func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("read_file() expects 1 argument")
		}
		if args[0].Type != ValString {
			return nil, fmt.Errorf("read_file() expects string path")
		}
		p := args[0].StrVal
		if err := validatePath(p); err != nil {
			return nil, err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read_file(%q): %s", p, err)
		}
		return strVal(string(data)), nil
	}
}

func (rt *Runtime) builtinWriteFile() func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
	return func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("write_file() expects 2 arguments")
		}
		if args[0].Type != ValString {
			return nil, fmt.Errorf("write_file() expects string path")
		}
		p := args[0].StrVal
		if err := validatePath(p); err != nil {
			return nil, err
		}
		if err := gov.checkAction("write_file", p, rt.file, 0, 0); err != nil {
			return nil, err
		}
		rt.traceLog("write_file:"+p, 0, 0)
		content := valueToString(args[1])
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("write_file(%q): %s", p, err)
		}
		return nilValue, nil
	}
}

func (rt *Runtime) builtinAppendFile() func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
	return func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("append_file() expects 2 arguments")
		}
		if args[0].Type != ValString {
			return nil, fmt.Errorf("append_file() expects string path")
		}
		p := args[0].StrVal
		if err := validatePath(p); err != nil {
			return nil, err
		}
		if err := gov.checkAction("append_file", p, rt.file, 0, 0); err != nil {
			return nil, err
		}
		rt.traceLog("append_file:"+p, 0, 0)
		content := valueToString(args[1])
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("append_file(%q): %s", p, err)
		}
		defer f.Close()
		if _, err := f.WriteString(content); err != nil {
			return nil, fmt.Errorf("append_file(%q): %s", p, err)
		}
		return nilValue, nil
	}
}

func (rt *Runtime) builtinHttpGet() func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
	return func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("http_get() expects at least 1 argument")
		}
		if args[0].Type != ValString {
			return nil, fmt.Errorf("http_get() expects string url")
		}
		rawURL := args[0].StrVal
		timeoutMs := 5000.0
		maxBytes := 20000
		if len(args) >= 2 && args[1].Type == ValNumber {
			timeoutMs, _ = args[1].NumVal.Float64()
		}
		if len(args) >= 3 && args[2].Type == ValNumber {
			f, _ := args[2].NumVal.Float64()
			maxBytes = int(f)
		}
		if err := gov.checkAction("http_get", rawURL, rt.file, 0, 0); err != nil {
			return nil, err
		}
		rt.traceLog("http_get:"+rawURL, 0, 0)
		client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
		resp, err := client.Get(rawURL)
		if err != nil {
			return nil, fmt.Errorf("http_get(%q): %s", rawURL, err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
		if err != nil {
			return nil, fmt.Errorf("http_get(%q): %s", rawURL, err)
		}
		return strVal(string(body)), nil
	}
}

func (rt *Runtime) builtinHttpPost() func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
	return func(args []*Value, gov *GovernanceState, rt2 *Runtime) (*Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("http_post() expects at least 2 arguments")
		}
		if args[0].Type != ValString {
			return nil, fmt.Errorf("http_post() expects string url")
		}
		rawURL := args[0].StrVal
		body := valueToString(args[1])
		timeoutMs := 5000.0
		maxBytes := 20000
		if len(args) >= 3 && args[2].Type == ValNumber {
			timeoutMs, _ = args[2].NumVal.Float64()
		}
		if len(args) >= 4 && args[3].Type == ValNumber {
			f, _ := args[3].NumVal.Float64()
			maxBytes = int(f)
		}
		if err := gov.checkAction("http_post", rawURL, rt.file, 0, 0); err != nil {
			return nil, err
		}
		rt.traceLog("http_post:"+rawURL, 0, 0)
		client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
		resp, err := client.Post(rawURL, "application/json", strings.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("http_post(%q): %s", rawURL, err)
		}
		defer resp.Body.Close()
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
		if err != nil {
			return nil, fmt.Errorf("http_post(%q): %s", rawURL, err)
		}
		return strVal(string(respBody)), nil
	}
}
