package filters

import "fmt"

func NewDefaultRegistry() Registry {
	return Registry{
		LabelFilterName:    LabelFilter,
		AgeFilterName:      AgeFilter,
		KorLabelFilterName: KorLabelFilter,
	}
}

// Registry is a collection of all available filters. The framework uses a
type Registry map[string]FilterFunc

func (r Registry) Register(name string, filter FilterFunc) error {
	if _, ok := r[name]; ok {
		return fmt.Errorf("a filter named %v already exists", name)
	}
	r[name] = filter
	return nil
}

func (r Registry) Unregister(name string) error {
	if _, ok := r[name]; !ok {
		return fmt.Errorf("no filter named %v exists", name)
	}
	delete(r, name)
	return nil
}

func (r Registry) Merge(in Registry) error {
	for name, filter := range in {
		if err := r.Register(name, filter); err != nil {
			return err
		}
	}
	return nil
}
