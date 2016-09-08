package remoteworkitem

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestFlatten(t *testing.T) {
	testString := []byte(`{"name":"shoubhik","assignee":{"1":"sbose","2":"pranav","3":{"4":"sbose4","5":"hjhj"}},"name":"shoubhikjjjhjh"}`)
	var nestedMap map[string]interface{}
	err := json.Unmarshal(testString, &nestedMap)

	if err != nil {
		t.Error("Incorrect dataset , %s ", testString)
	}

	flattendedMap := Flatten(nestedMap)
	fmt.Println(flattendedMap)
}
