package util

import (
        "fmt"
        "os"
        "github.com/goccy/go-yaml"
)

func ParseValuesFile(valuesFile string, source string) (map[string]any, error){
        valuesBytes, err := os.ReadFile(valuesFile)
        if err != nil {
                return nil, fmt.Errorf("could not read %s values file: %w", source, err)
        }
        var valuesMap map[string]any
        yaml.Unmarshal(valuesBytes, &valuesMap)
	return valuesMap, nil
}
