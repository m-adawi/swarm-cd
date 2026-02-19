package util

// ExternalResolver is a pluggable resolver for external values.
// It is intended for resolving any value (not only secrets) from an external
// source based on a special reference format.
//
// Resolve should return:
//   - value: resolved string value
//   - isRef: true if the given ref belongs to this resolver (even if error)
//   - err:   non-nil if resolution failed
//
// If isRef is false, the resolver must ignore the ref and leave it to others.
type ExternalResolver interface {
	Resolve(ref string) (value string, isRef bool, err error)
}

var externalResolvers []ExternalResolver

// RegisterExternalResolver adds a new resolver to the global chain.
// It is safe to call this from package init / initialization code.
func RegisterExternalResolver(r ExternalResolver) {
	externalResolvers = append(externalResolvers, r)
}

// ResolveExternalReference tries all registered resolvers in order until one of
// them claims the reference (isRef == true). If none of them recognize it,
// ("", false, nil) is returned.
func ResolveExternalReference(ref string) (string, bool, error) {
	for _, r := range externalResolvers {
		val, isRef, err := r.Resolve(ref)
		if isRef {
			return val, true, err
		}
	}
	return "", false, nil
}

// InitExternalResolvers initializes all configured external resolvers.
// This is the single entrypoint that higher-level packages (like cmd/)
// should call, without depending on concrete implementations (Vault, etc.).
func InitExternalResolvers() error {
	// Initialize Vault-based resolver (if configured).
	if err := InitVault(); err != nil {
		return err
	}

	// Initialize Consul-based resolver (if configured).
	if err := InitConsul(); err != nil {
		return err
	}

	return nil
}

