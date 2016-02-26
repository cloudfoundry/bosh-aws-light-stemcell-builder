package ec2ami

import (
	"sync"

	"gopkg.in/yaml.v2"
)

func NewCollection() *Collection {
	return &Collection{amis: make(map[string]Info)}
}

type Collection struct {
	sync.Mutex
	amis map[string]Info
}

func (a *Collection) Add(region string, amiInfo Info) {
	a.Lock()
	defer a.Unlock()

	a.amis[region] = amiInfo
}

func (a *Collection) Get(region string) Info {
	a.Lock()
	defer a.Unlock()

	return a.amis[region]
}

func (a *Collection) GetAll() map[string]Info {
	a.Lock()
	defer a.Unlock()

	return a.amis
}

func (a *Collection) MarshalYAML() (interface{}, error) {
	var marshaledAmis yaml.MapSlice
	for region, info := range a.GetAll() {
		marshaledAmis = append(marshaledAmis, yaml.MapItem{Key: region, Value: info.AmiID})
	}
	return marshaledAmis, nil
}
