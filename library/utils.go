package library

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func PrettyPrint(k string, v interface{}) {
	b, _ := json.Marshal(v)
	var out bytes.Buffer
	_ = json.Indent(&out, b, "", "\t")
	fmt.Println(k, out.String())
}
