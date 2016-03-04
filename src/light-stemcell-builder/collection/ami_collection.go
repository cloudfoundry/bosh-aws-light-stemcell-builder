package collection

import (
	"light-stemcell-builder/resources"
	"sync"
)

type Ami struct {
	VirtualizationType string
	sync.Mutex
	amis []resources.Ami
}

func (a *Ami) Add(ami resources.Ami) {
	a.Lock()
	defer a.Unlock()

	a.amis = append(a.amis, ami)
}

func (a *Ami) GetAll() []resources.Ami {
	a.Lock()
	defer a.Unlock()

	return a.amis
}

func (a *Ami) Merge(other *Ami) {
	for _, otherAmi := range other.GetAll() {
		a.Add(otherAmi)
	}
}
