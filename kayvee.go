package kayvee

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Format converts a map to a string of space-delimited key=val pairs
func Format(data map[string]interface{}) string {
	keyVals := []string{}
	for key, val := range data {
		formattedVal, _ := json.Marshal(val)
		formatted := fmt.Sprintf("%s=%v", key, string(formattedVal))
		keyVals = append(keyVals, formatted)
	}
	sort.Strings(keyVals)
	return strings.Join(keyVals, " ")
}

// FormatLog is similar to Format, but takes additional reserved params to promote logging best-practices
func FormatLog(source string, level string, title string, data map[string]interface{}) string {
	formattedSource := Format(map[string]interface{}{"source": source})
	formattedLevel := Format(map[string]interface{}{"level": level})
	formattedTitle := Format(map[string]interface{}{"title": title})
	formattedData := Format(data)

	allStrings := []string{formattedSource, formattedLevel, formattedTitle}
	if len(formattedData) > 0 {
		allStrings = append(allStrings, formattedData)
	}
	return strings.Join(allStrings, " ")
}
