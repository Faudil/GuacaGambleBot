package universe

var registry = map[string]*Definition{}

func Register(u *Definition) {
	registry[u.ID] = u
}

func Get(id string) *Definition {
	return registry[id]
}

func List() []*Definition {
	out := make([]*Definition, 0, len(registry))
	for _, u := range registry {
		out = append(out, u)
	}
	return out
}
