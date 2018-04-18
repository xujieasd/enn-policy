package testing

import (
	utilIPSet "enn-policy/pkg/util/ipset"
	"fmt"
	"strings"
	"reflect"
)

type Faker struct {
	FakeSet  map[*utilIPSet.IPSet][]*utilIPSet.Entry
}

func NewFaker() *Faker{
	return &Faker{
		FakeSet:    make(map[*utilIPSet.IPSet][]*utilIPSet.Entry),
	}
}

func (f *Faker) FlushAll() error{

	for ipset, _ := range f.FakeSet{
		emptyEntries := make([]*utilIPSet.Entry, 0)
		f.FakeSet[ipset] = emptyEntries
	}

	return nil
}

func (f *Faker) DestroyAll() error{

	f.FakeSet = make(map[*utilIPSet.IPSet][]*utilIPSet.Entry)

	return nil
}

func (f *Faker) FlushIPSet(set *utilIPSet.IPSet) error{

	emptyEntries := make([]*utilIPSet.Entry, 0)
	f.FakeSet[set] = emptyEntries

	return nil
}

func (f *Faker) CreateIPSet(set *utilIPSet.IPSet, ignoreExitErr bool) error{

	emptyEntries := make([]*utilIPSet.Entry, 0)
	f.FakeSet[set] = emptyEntries

	return nil
}

func (f *Faker) DestroyIPSet(set *utilIPSet.IPSet) error{

	_, ok := f.FakeSet[set]
	if !ok{
		return fmt.Errorf("cannot find ipset:%s in fake map", set.Name)
	}
	delete(f.FakeSet, set)

	return nil
}

func (f *Faker) ListIPSetsName() ([]string, error){

	result := make([]string, 0)
	for ipset,_ := range f.FakeSet{
		result = append(result, ipset.Name)
	}

	return result, nil
}

func (f *Faker) GetIPSet(name string) (*utilIPSet.IPSet, error){

	for ipset,_ := range f.FakeSet{
		if strings.Compare(ipset.Name, name) == 0{
			return ipset, nil
		}
	}

	return nil, fmt.Errorf("cannot find ipset:%s in fake map", name)
}

func (f *Faker) AddEntry(set *utilIPSet.IPSet, entry *utilIPSet.Entry, ignoreExitErr bool) error{

	entries, ok := f.FakeSet[set]
	if !ok{
		return fmt.Errorf("cannot find ipset:%s in fake map", set.Name)
	}

	for _, fakeEntry := range entries{
		if reflect.DeepEqual(fakeEntry, entry){
			return nil
		}
	}
	entries = append(entries, entry)
	f.FakeSet[set] = entries
	return nil
}

func (f *Faker) DelEntry(set *utilIPSet.IPSet, entry *utilIPSet.Entry, ignoreExitErr bool) error{

	entries, ok := f.FakeSet[set]
	if !ok{
		return fmt.Errorf("cannot find ipset:%s in fake map", set.Name)
	}

	active := false
	var target int
	for target = range entries{
		if reflect.DeepEqual(entries[target], entry){
			active = true
			break
		}
	}
	if !active{
		fmt.Errorf("cannot find entry type:%s ip:%s net:%s in ipset:%s", entry.Type, entry.IP, entry.Net, set.Name)
	}

	f.FakeSet[set] = append(f.FakeSet[set][:target],f.FakeSet[set][target+1:]...)

	return nil
}

func (f *Faker) ListEntry(set *utilIPSet.IPSet) ([]*utilIPSet.Entry, error){

	entries, ok := f.FakeSet[set]
	if !ok{
		return nil, fmt.Errorf("cannot find ipset:%s in fake map", set.Name)
	}

	return entries, nil
}