// Code generated by counterfeiter. DO NOT EDIT.
package driversetfakes

import (
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"
	"sync"
)

type FakeIsolatedRegionDriverSet struct {
	CreateAmiDriverStub        func() resources.AmiDriver
	createAmiDriverMutex       sync.RWMutex
	createAmiDriverArgsForCall []struct {
	}
	createAmiDriverReturns struct {
		result1 resources.AmiDriver
	}
	createAmiDriverReturnsOnCall map[int]struct {
		result1 resources.AmiDriver
	}
	CreateSnapshotDriverStub        func() resources.SnapshotDriver
	createSnapshotDriverMutex       sync.RWMutex
	createSnapshotDriverArgsForCall []struct {
	}
	createSnapshotDriverReturns struct {
		result1 resources.SnapshotDriver
	}
	createSnapshotDriverReturnsOnCall map[int]struct {
		result1 resources.SnapshotDriver
	}
	MachineImageDriverStub        func() resources.MachineImageDriver
	machineImageDriverMutex       sync.RWMutex
	machineImageDriverArgsForCall []struct {
	}
	machineImageDriverReturns struct {
		result1 resources.MachineImageDriver
	}
	machineImageDriverReturnsOnCall map[int]struct {
		result1 resources.MachineImageDriver
	}
	VolumeDriverStub        func() resources.VolumeDriver
	volumeDriverMutex       sync.RWMutex
	volumeDriverArgsForCall []struct {
	}
	volumeDriverReturns struct {
		result1 resources.VolumeDriver
	}
	volumeDriverReturnsOnCall map[int]struct {
		result1 resources.VolumeDriver
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeIsolatedRegionDriverSet) CreateAmiDriver() resources.AmiDriver {
	fake.createAmiDriverMutex.Lock()
	ret, specificReturn := fake.createAmiDriverReturnsOnCall[len(fake.createAmiDriverArgsForCall)]
	fake.createAmiDriverArgsForCall = append(fake.createAmiDriverArgsForCall, struct {
	}{})
	stub := fake.CreateAmiDriverStub
	fakeReturns := fake.createAmiDriverReturns
	fake.recordInvocation("CreateAmiDriver", []interface{}{})
	fake.createAmiDriverMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeIsolatedRegionDriverSet) CreateAmiDriverCallCount() int {
	fake.createAmiDriverMutex.RLock()
	defer fake.createAmiDriverMutex.RUnlock()
	return len(fake.createAmiDriverArgsForCall)
}

func (fake *FakeIsolatedRegionDriverSet) CreateAmiDriverCalls(stub func() resources.AmiDriver) {
	fake.createAmiDriverMutex.Lock()
	defer fake.createAmiDriverMutex.Unlock()
	fake.CreateAmiDriverStub = stub
}

func (fake *FakeIsolatedRegionDriverSet) CreateAmiDriverReturns(result1 resources.AmiDriver) {
	fake.createAmiDriverMutex.Lock()
	defer fake.createAmiDriverMutex.Unlock()
	fake.CreateAmiDriverStub = nil
	fake.createAmiDriverReturns = struct {
		result1 resources.AmiDriver
	}{result1}
}

func (fake *FakeIsolatedRegionDriverSet) CreateAmiDriverReturnsOnCall(i int, result1 resources.AmiDriver) {
	fake.createAmiDriverMutex.Lock()
	defer fake.createAmiDriverMutex.Unlock()
	fake.CreateAmiDriverStub = nil
	if fake.createAmiDriverReturnsOnCall == nil {
		fake.createAmiDriverReturnsOnCall = make(map[int]struct {
			result1 resources.AmiDriver
		})
	}
	fake.createAmiDriverReturnsOnCall[i] = struct {
		result1 resources.AmiDriver
	}{result1}
}

func (fake *FakeIsolatedRegionDriverSet) CreateSnapshotDriver() resources.SnapshotDriver {
	fake.createSnapshotDriverMutex.Lock()
	ret, specificReturn := fake.createSnapshotDriverReturnsOnCall[len(fake.createSnapshotDriverArgsForCall)]
	fake.createSnapshotDriverArgsForCall = append(fake.createSnapshotDriverArgsForCall, struct {
	}{})
	stub := fake.CreateSnapshotDriverStub
	fakeReturns := fake.createSnapshotDriverReturns
	fake.recordInvocation("CreateSnapshotDriver", []interface{}{})
	fake.createSnapshotDriverMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeIsolatedRegionDriverSet) CreateSnapshotDriverCallCount() int {
	fake.createSnapshotDriverMutex.RLock()
	defer fake.createSnapshotDriverMutex.RUnlock()
	return len(fake.createSnapshotDriverArgsForCall)
}

func (fake *FakeIsolatedRegionDriverSet) CreateSnapshotDriverCalls(stub func() resources.SnapshotDriver) {
	fake.createSnapshotDriverMutex.Lock()
	defer fake.createSnapshotDriverMutex.Unlock()
	fake.CreateSnapshotDriverStub = stub
}

func (fake *FakeIsolatedRegionDriverSet) CreateSnapshotDriverReturns(result1 resources.SnapshotDriver) {
	fake.createSnapshotDriverMutex.Lock()
	defer fake.createSnapshotDriverMutex.Unlock()
	fake.CreateSnapshotDriverStub = nil
	fake.createSnapshotDriverReturns = struct {
		result1 resources.SnapshotDriver
	}{result1}
}

func (fake *FakeIsolatedRegionDriverSet) CreateSnapshotDriverReturnsOnCall(i int, result1 resources.SnapshotDriver) {
	fake.createSnapshotDriverMutex.Lock()
	defer fake.createSnapshotDriverMutex.Unlock()
	fake.CreateSnapshotDriverStub = nil
	if fake.createSnapshotDriverReturnsOnCall == nil {
		fake.createSnapshotDriverReturnsOnCall = make(map[int]struct {
			result1 resources.SnapshotDriver
		})
	}
	fake.createSnapshotDriverReturnsOnCall[i] = struct {
		result1 resources.SnapshotDriver
	}{result1}
}

func (fake *FakeIsolatedRegionDriverSet) MachineImageDriver() resources.MachineImageDriver {
	fake.machineImageDriverMutex.Lock()
	ret, specificReturn := fake.machineImageDriverReturnsOnCall[len(fake.machineImageDriverArgsForCall)]
	fake.machineImageDriverArgsForCall = append(fake.machineImageDriverArgsForCall, struct {
	}{})
	stub := fake.MachineImageDriverStub
	fakeReturns := fake.machineImageDriverReturns
	fake.recordInvocation("MachineImageDriver", []interface{}{})
	fake.machineImageDriverMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeIsolatedRegionDriverSet) MachineImageDriverCallCount() int {
	fake.machineImageDriverMutex.RLock()
	defer fake.machineImageDriverMutex.RUnlock()
	return len(fake.machineImageDriverArgsForCall)
}

func (fake *FakeIsolatedRegionDriverSet) MachineImageDriverCalls(stub func() resources.MachineImageDriver) {
	fake.machineImageDriverMutex.Lock()
	defer fake.machineImageDriverMutex.Unlock()
	fake.MachineImageDriverStub = stub
}

func (fake *FakeIsolatedRegionDriverSet) MachineImageDriverReturns(result1 resources.MachineImageDriver) {
	fake.machineImageDriverMutex.Lock()
	defer fake.machineImageDriverMutex.Unlock()
	fake.MachineImageDriverStub = nil
	fake.machineImageDriverReturns = struct {
		result1 resources.MachineImageDriver
	}{result1}
}

func (fake *FakeIsolatedRegionDriverSet) MachineImageDriverReturnsOnCall(i int, result1 resources.MachineImageDriver) {
	fake.machineImageDriverMutex.Lock()
	defer fake.machineImageDriverMutex.Unlock()
	fake.MachineImageDriverStub = nil
	if fake.machineImageDriverReturnsOnCall == nil {
		fake.machineImageDriverReturnsOnCall = make(map[int]struct {
			result1 resources.MachineImageDriver
		})
	}
	fake.machineImageDriverReturnsOnCall[i] = struct {
		result1 resources.MachineImageDriver
	}{result1}
}

func (fake *FakeIsolatedRegionDriverSet) VolumeDriver() resources.VolumeDriver {
	fake.volumeDriverMutex.Lock()
	ret, specificReturn := fake.volumeDriverReturnsOnCall[len(fake.volumeDriverArgsForCall)]
	fake.volumeDriverArgsForCall = append(fake.volumeDriverArgsForCall, struct {
	}{})
	stub := fake.VolumeDriverStub
	fakeReturns := fake.volumeDriverReturns
	fake.recordInvocation("VolumeDriver", []interface{}{})
	fake.volumeDriverMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeIsolatedRegionDriverSet) VolumeDriverCallCount() int {
	fake.volumeDriverMutex.RLock()
	defer fake.volumeDriverMutex.RUnlock()
	return len(fake.volumeDriverArgsForCall)
}

func (fake *FakeIsolatedRegionDriverSet) VolumeDriverCalls(stub func() resources.VolumeDriver) {
	fake.volumeDriverMutex.Lock()
	defer fake.volumeDriverMutex.Unlock()
	fake.VolumeDriverStub = stub
}

func (fake *FakeIsolatedRegionDriverSet) VolumeDriverReturns(result1 resources.VolumeDriver) {
	fake.volumeDriverMutex.Lock()
	defer fake.volumeDriverMutex.Unlock()
	fake.VolumeDriverStub = nil
	fake.volumeDriverReturns = struct {
		result1 resources.VolumeDriver
	}{result1}
}

func (fake *FakeIsolatedRegionDriverSet) VolumeDriverReturnsOnCall(i int, result1 resources.VolumeDriver) {
	fake.volumeDriverMutex.Lock()
	defer fake.volumeDriverMutex.Unlock()
	fake.VolumeDriverStub = nil
	if fake.volumeDriverReturnsOnCall == nil {
		fake.volumeDriverReturnsOnCall = make(map[int]struct {
			result1 resources.VolumeDriver
		})
	}
	fake.volumeDriverReturnsOnCall[i] = struct {
		result1 resources.VolumeDriver
	}{result1}
}

func (fake *FakeIsolatedRegionDriverSet) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createAmiDriverMutex.RLock()
	defer fake.createAmiDriverMutex.RUnlock()
	fake.createSnapshotDriverMutex.RLock()
	defer fake.createSnapshotDriverMutex.RUnlock()
	fake.machineImageDriverMutex.RLock()
	defer fake.machineImageDriverMutex.RUnlock()
	fake.volumeDriverMutex.RLock()
	defer fake.volumeDriverMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeIsolatedRegionDriverSet) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ driverset.IsolatedRegionDriverSet = new(FakeIsolatedRegionDriverSet)