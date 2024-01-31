package filters

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func NewNormalFramework(r Registry) Framework {
	return &normalFramework{registry: r}
}

type normalFramework struct {
	registry Registry
	object   runtime.Object
}

func (n *normalFramework) Run(opts *Options, disable ...string) (bool, error) {
	for name, f := range n.registry {
		if isIn(name, disable) {
			continue
		}
		if f(n.object, opts) {
			return true, nil
		}
	}
	return false, nil
}

func (n *normalFramework) RunFilter(name string, opts *Options) (bool, error) {
	f, ok := n.registry[name]
	if !ok {
		return true, nil
	}
	return f(n.object, opts), nil

}

func (n *normalFramework) AddFilter(name string, f FilterFunc) Framework {
	out := n.DeepCopy()
	_ = out.registry.Register(name, f)
	return out
}

func (n *normalFramework) SetObject(object runtime.Object) Framework {
	out := n.DeepCopy()
	out.object = object
	return out
}

func (n *normalFramework) SetRegistry(r Registry) Framework {
	out := n.DeepCopy()
	out.registry = r
	return out
}

func (n *normalFramework) DeepCopy() *normalFramework {
	out := new(normalFramework)
	n.DeepCopyInto(out)
	return out
}

func (n *normalFramework) DeepCopyInto(out *normalFramework) {
	*out = *n
	if n.registry != nil {
		in, out := &n.registry, &out.registry
		*out = make(Registry, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

func isIn(name string, disable []string) bool {
	for _, v := range disable {
		if v == name {
			return true
		}
	}
	return false
}
