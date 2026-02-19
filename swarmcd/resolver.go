package swarmcd

import (
	"fmt"
	"strings"

	"github.com/m-adawi/swarm-cd/util"
)

// resolveExternalValues processes environment variables in the compose map
// and resolves any external references (vault:, consul:, etc.) by calling
// the appropriate resolver.
func resolveExternalValues(composeMap map[string]any) error {
	services, ok := composeMap["services"].(map[string]any)
	if !ok {
		return nil
	}

	for serviceName, svc := range services {
		svcMap, ok := svc.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid compose file: service %s must be a map", serviceName)
		}

		env, ok := svcMap["environment"]
		if !ok {
			continue
		}

		switch typed := env.(type) {
		case map[string]any:
			for k, v := range typed {
				strVal, ok := v.(string)
				if !ok {
					continue
				}
				resolved, isRef, err := util.ResolveExternalReference(strVal)
				if err != nil {
					return fmt.Errorf("service %s env %s: %w", serviceName, k, err)
				}
				if isRef {
					typed[k] = resolved
				}
			}
		case []any:
			for i, item := range typed {
				strItem, ok := item.(string)
				if !ok {
					continue
				}
				parts := strings.SplitN(strItem, "=", 2)
				if len(parts) != 2 {
					continue
				}
				key := parts[0]
				val := parts[1]
				resolved, isRef, err := util.ResolveExternalReference(val)
				if err != nil {
					return fmt.Errorf("service %s env %s: %w", serviceName, key, err)
				}
				if isRef {
					typed[i] = key + "=" + resolved
				}
			}
		default:
			// unknown env format, ignore
			continue
		}
	}

	return nil
}
