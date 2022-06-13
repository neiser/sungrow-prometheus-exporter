package config

import "gopkg.in/yaml.v3"

type namer interface {
	getName() string
}

func unmarshalSequenceToMap[T namer](node *yaml.Node, result *map[string]*T) error {
	var s []*T
	err := node.Decode(&s)
	if err != nil {
		return err
	}
	*result = make(map[string]*T)
	for _, metric := range s {
		(*result)[(*metric).getName()] = metric
	}
	return nil
}
