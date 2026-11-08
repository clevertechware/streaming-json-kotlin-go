package export

import (
	"encoding/json"
	"fmt"
	"strings"
)

func streamOf(count int) string {
	items := make([]string, count)
	for i := range items {
		items[i] = fmt.Sprintf(`{"id":"MV-%05d","warehouseId":"WH-1","code":"C-%05d","label":"Mouvement %d"}`, i, i, i)
	}
	return `{"results":[` + strings.Join(items, ",") + `]}`
}

func mustMarshal(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
