package testing

import (
	"testing"
	"enn-policy/pkg/util/ipset"
)

var testIPSet = []*ipset.IPSet{
	{
		Name:      "testSet1",
		Type:      ipset.TypeHashIP,
		Family:    ipset.FamilyIPV4,
		HashSize:  ipset.DefaultHashSize,
		MaxElem:   ipset.DefaultMaxElem,
		Reference: 0,
	},
	{
		Name:      "testSet2",
		Type:      ipset.TypeHashNet,
		Family:    ipset.FamilyIPV4,
		HashSize:  ipset.DefaultHashSize,
		MaxElem:   ipset.DefaultMaxElem,
		Reference: 0,
	},
	{
		Name:      "testSet3",
		Type:      ipset.TypeHashIPPort,
		Family:    ipset.FamilyIPV4,
		HashSize:  ipset.DefaultHashSize,
		MaxElem:   ipset.DefaultMaxElem,
		Reference: 0,
	},
	{
		Name:      "testSet4",
		Type:      ipset.TypeHashNetPort,
		Family:    ipset.FamilyIPV4,
		HashSize:  ipset.DefaultHashSize,
		MaxElem:   ipset.DefaultMaxElem,
		Reference: 0,
	},
}

var testEntry = [][]*ipset.Entry{
	{
		{
			Type: ipset.TypeHashIP,
			IP:   "10.0.0.1",
		},
		{
			Type: ipset.TypeHashIP,
			IP:   "10.0.0.2",
		},
		{
			Type: ipset.TypeHashIP,
			IP:   "10.0.0.3",
		},
	},
	{
		{
			Type: ipset.TypeHashNet,
			Net:  "10.0.0.0/24",
		},
		{
			Type: ipset.TypeHashNet,
			Net:  "10.0.1.0/24",
		},
		{
			Type: ipset.TypeHashNet,
			Net:  "10.0.2.0/24",
		},
	},
	{
		{
			Type: ipset.TypeHashIPPort,
			IP:   "10.0.0.1",
			Port: "80",
		},
		{
			Type: ipset.TypeHashIPPort,
			IP:   "10.0.0.2",
			Port: "81",
		},
		{
			Type: ipset.TypeHashIPPort,
			IP:   "10.0.0.3",
			Port: "82",
		},
	},
	{
		{
			Type: ipset.TypeHashNetPort,
			Net:  "10.0.0.0/24",
			Port: "80",
		},
		{
			Type: ipset.TypeHashNetPort,
			Net:  "10.0.1.0/24",
			Port: "81",
		},
		{
			Type: ipset.TypeHashNetPort,
			Net:  "10.0.2.0/24",
			Port: "82",
		},
	},
}

func TestFakeIPSet(t *testing.T){

	ipset := NewFaker()

	for i:= 0; i<4; i++{
		err := ipset.CreateIPSet(testIPSet[i],true)
		if err != nil{
			t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[i].Name, testIPSet[i].Type, err)
		}

	}

	for i:= 0; i<4; i++{

		kernelSet, err := ipset.GetIPSet(testIPSet[i].Name)
		if err != nil {
			t.Errorf("get ipset fail name:%s type:%s err:%v", testIPSet[i].Name, testIPSet[i].Type, err)
		}
		isCorrectSet(t,testIPSet[i],kernelSet)

	}

	setname, err := ipset.ListIPSetsName()
	if err!= nil{
		t.Errorf("list ipset fail %v", err)
	} else {
		if len(setname) != 4{
			t.Errorf("invaild set lenth %d expect 4", len(setname))
		}

		var testNames []string
		for _, ipset := range testIPSet{
			testNames = append(testNames, ipset.Name)
		}

		for _, name := range setname{
			find := false
			for _, ipset := range testNames{
				if name == ipset{
					find = true
				}
			}

			if !find{
				t.Errorf("cannot find set name %s", name)
			}
		}
	}

	for i:= 0; i<4; i++{
		ipset.DestroyIPSet(testIPSet[i])
	}

	if len(ipset.FakeSet) != 0 {
		t.Errorf("number of ipset error: %d after delete, expected: %d", len(ipset.FakeSet), 0)
	}
}

func TestFakeEntry(t *testing.T){

	ipset := NewFaker()

	for i:= 0; i<4; i++{
		err := ipset.CreateIPSet(testIPSet[i],true)
		if err != nil{
			t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[i].Name, testIPSet[i].Type, err)
			continue
		}

		for j:= 0; j<3; j++{
			err := ipset.AddEntry(testIPSet[i],testEntry[i][j],true)
			if err!= nil{
				t.Errorf("add entry %s to ipset %s,%s error: %v", testEntry[i][j].IP, testIPSet[i].Name, testIPSet[i].Type, err)
			}
		}
	}

	if len(ipset.FakeSet) != 4 {
		t.Errorf("number of ipset is not correct: %d, expect %d", len(ipset.FakeSet), 4)
	}

	for fakeSet, fakeEntries := range ipset.FakeSet{
		if len(fakeEntries) != 3 {
			t.Errorf("number of ipset: %s is not correct: %d, expect %d", fakeSet.Name, len(fakeEntries), 3)
		}
	}

	for i:= 0; i<4; i++{
		err := ipset.DelEntry(testIPSet[i], testEntry[i][2], false)
		if err != nil{
			t.Errorf("delete entry fail")
		}
	}

	for fakeSet, fakeEntries := range ipset.FakeSet{
		if len(fakeEntries) != 2 {
			t.Errorf("number of ipset: %s is not correct: %d, expect %d", fakeSet.Name, len(fakeEntries), 2)
		}
	}

	for i:= 0; i<4; i++ {
		kernelEntry, err := ipset.ListEntry(testIPSet[i])
		if err != nil {
			t.Errorf("list entry for set %s:%s error", testIPSet[i].Name, testIPSet[i].Type)
		} else {
			isCorrectEntry(t, testEntry[i], kernelEntry, testIPSet[i].Type, 2)
		}
	}

}

func isCorrectSet(t *testing.T, testIPSet *ipset.IPSet, kernelSet *ipset.IPSet){

	if testIPSet.Name != kernelSet.Name{
		t.Errorf("set name is not correct %s, expect %s", kernelSet.Name, testIPSet.Name)
	}
	if testIPSet.Type != kernelSet.Type{
		t.Errorf("set type is not correct %s, expect %s", kernelSet.Type, testIPSet.Type)
	}
	if testIPSet.MaxElem != kernelSet.MaxElem{
		t.Errorf("set maxElem is not correct %d, expect %d", kernelSet.MaxElem, testIPSet.MaxElem)
	}
	if testIPSet.HashSize != kernelSet.HashSize{
		t.Errorf("set hashSize is not correct %d, expect %d", kernelSet.HashSize, testIPSet.HashSize)
	}
	if testIPSet.Family != kernelSet.Family{
		t.Errorf("set family is not correct %s, expect %s", kernelSet.Family, testIPSet.Family)
	}
	if testIPSet.Reference != kernelSet.Reference{
		t.Errorf("set reference is not correct %d, expect %d", kernelSet.Reference, testIPSet.Reference)
	}
}

func isCorrectEntry(t *testing.T, testEntry []*ipset.Entry, kernelEntry []*ipset.Entry, entryType string, length int){

	if length != len(kernelEntry){
		t.Errorf("entry number is not correct %d, expect %d", len(kernelEntry), length)
		return
	}

	for _, entry := range kernelEntry{
		find := false
		for _, t_entry := range testEntry{
			if entry.Type != entryType{
				t.Errorf("invalid entryType %s, expected %s", entryType, entry.Type)
			}
			switch  entryType{
			case ipset.TypeHashIP:
				if entry.IP == t_entry.IP{
					find = true
					break
				}
			case ipset.TypeHashNet:
				if entry.Net == t_entry.Net{
					find = true
					break
				}
			case ipset.TypeHashIPPort:
				if entry.IP == t_entry.IP && entry.Port == t_entry.Port{
					find = true
					break
				}
			case ipset.TypeHashNetPort:
				if entry.Net == t_entry.Net && entry.Port == t_entry.Port{
					find = true
					break
				}
			}
		}
		if !find{
			t.Errorf("can not find expect entry ip:%s port:%s net:%s type:%s", entry.IP, entry.Port, entry.Net, entryType)
		}
	}
}