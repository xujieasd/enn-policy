package ipset

// Attention: this test code will do ipset operation in your host machine
// If your host machine already has ipset data, this test code maybe impact your old ipset data
// So please backups old ipset data before running this test code (e.g ipset save)

import (
	"testing"
	"k8s.io/utils/exec"
)

var testIPSet = []*IPSet{
	{
		Name:      "testSet1",
		Type:      TypeHashIP,
		Family:    FamilyIPV4,
		HashSize:  DefaultHashSize,
		MaxElem:   DefaultMaxElem,
		Reference: 0,
	},
	{
		Name:      "testSet2",
		Type:      TypeHashNet,
		Family:    FamilyIPV4,
		HashSize:  DefaultHashSize,
		MaxElem:   DefaultMaxElem,
		Reference: 0,
	},
	{
		Name:      "testSet3",
		Type:      TypeHashIPPort,
		Family:    FamilyIPV4,
		HashSize:  DefaultHashSize,
		MaxElem:   DefaultMaxElem,
		Reference: 0,
	},
	{
		Name:      "testSet4",
		Type:      TypeHashNetPort,
		Family:    FamilyIPV4,
		HashSize:  DefaultHashSize,
		MaxElem:   DefaultMaxElem,
		Reference: 0,
	},
}

func TestEnnIPSet_CreateIPSet(t *testing.T){

	execer := exec.New()
	ipset := NewEnnIPSet(execer)


	for i:= 0; i<4; i++{
		err := ipset.CreateIPSet(testIPSet[i],true)
		if err != nil{
			t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[i].Name, testIPSet[i].Type, err)
		}

	}

	for i:= 0; i<4; i++{
		ipset.DestroyIPSet(testIPSet[i])
	}
}

func TestEnnIPSet_GetIPSet(t *testing.T) {

	execer := exec.New()
	ipset := NewEnnIPSet(execer)

	for i:= 0; i<4; i++{

		err := ipset.CreateIPSet(testIPSet[i],true)
		if err != nil{
			t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[i].Name, testIPSet[i].Type, err)
		} else{
			kernelSet, err := ipset.GetIPSet(testIPSet[i].Name)
			if err != nil {
				t.Errorf("get ipset fail name:%s type:%s err:%v", testIPSet[i].Name, testIPSet[i].Type, err)
			}
			isCorrectSet(t,testIPSet[i],kernelSet)
		}
	}

	for i:= 0; i<4; i++{
		ipset.DestroyIPSet(testIPSet[i])
	}

}

func TestEnnIPSet_ListIPSetsName(t *testing.T) {

	execer := exec.New()
	ipset := NewEnnIPSet(execer)

	setBeforeCreate, err := ipset.ListIPSetsName()
	if err!= nil{
		t.Errorf("list ipset fail %v", err)
	}

	for i:= 0; i<4; i++{
		err := ipset.CreateIPSet(testIPSet[i],true)
		if err != nil{
			t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[i].Name, testIPSet[i].Type, err)
		}
	}

	setname, err := ipset.ListIPSetsName()
	if err!= nil{
		t.Errorf("list ipset fail %v", err)
	} else {
		if len(setname) - len(setBeforeCreate) != 4{
			t.Errorf("invaild set lenth %d expect 4", len(setname) - len(setBeforeCreate))
		}

		var testNames []string
		for _, ipset := range testIPSet{
			testNames = append(testNames, ipset.Name)
		}

		for _, name := range setname{
			find := isFindSetName(name, testNames)
			if !find{
				t.Errorf("cannot find set name %s", name)
			}
		}
	}

	for i:= 0; i<4; i++{
		ipset.DestroyIPSet(testIPSet[i])
	}
}

func TestEnnIPSet_DestroyIPSet(t *testing.T){

	execer := exec.New()
	ipset := NewEnnIPSet(execer)

	for i:= 0; i<4; i++{
		err := ipset.CreateIPSet(testIPSet[i],true)
		if err != nil{
			t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[i].Name, testIPSet[i].Type, err)
		}
	}

	setname, err := ipset.ListIPSetsName()
	if err!= nil{
		t.Errorf("list ipset fail %v", err)
	}

	for i:= 0; i<4; i++{
		err := ipset.DestroyIPSet(testIPSet[i])
		if err != nil{
			t.Errorf("destroy ipset fail name:%s type:%s err:%v", testIPSet[i].Name, testIPSet[i].Type, err)
		}

		setDestroy, err := ipset.ListIPSetsName()
		if err!= nil{
			t.Errorf("list ipset fail %v", err)
		} else {
			length := len(setname) - len(setDestroy)
			if length != i+1{
				t.Errorf("invalid set lenth %d expect %d", length, i+1)
			}

			find := isFindSetName(testIPSet[i].Name, setDestroy)
			if find{
				t.Errorf("find unexpect set name %s", testIPSet[i].Name)
			}
		}
	}

}

func TestEnnIPSet_AddEntry(t *testing.T) {

	execer := exec.New()
	ipset := NewEnnIPSet(execer)


	err := ipset.CreateIPSet(testIPSet[0],true)
	if err != nil{
		t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[0].Name, testIPSet[0].Type, err)
	}

	testEntry1 := []*Entry{
		{
			Type: TypeHashIP,
			IP:   "10.0.0.1",
		},
		{
			Type: TypeHashIP,
			IP:   "10.0.0.2",
		},
		{
			Type: TypeHashIP,
			IP:   "10.0.0.3",
		},
	}

	for i:= 0; i<3; i++{
		err := ipset.AddEntry(testIPSet[0],testEntry1[i],true)
		if err!= nil{
			t.Errorf("add entry %s to ipset %s,%s error: %v", testEntry1[i].IP, testIPSet[0].Name, testIPSet[0].Type, err)
		}
	}


	err = ipset.CreateIPSet(testIPSet[1],true)
	if err != nil{
		t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[1].Name, testIPSet[1].Type, err)
	}

	testEntry2 := []*Entry{
		{
			Type: TypeHashNet,
			Net:  "10.0.0.0/24",
		},
		{
			Type: TypeHashNet,
			Net:  "10.0.1.0/24",
		},
		{
			Type: TypeHashNet,
			Net:  "10.0.2.0/24",
		},
	}

	for i:= 0; i<3; i++{
		err := ipset.AddEntry(testIPSet[1],testEntry2[i],true)
		if err!= nil{
			t.Errorf("add entry %s to ipset %s,%s error: %v", testEntry2[i].Net, testIPSet[1].Name, testIPSet[1].Type, err)
		}
	}

	err = ipset.CreateIPSet(testIPSet[2],true)
	if err != nil{
		t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[2].Name, testIPSet[2].Type, err)
	}

	testEntry3 := []*Entry{
		{
			Type: TypeHashIPPort,
			IP:   "10.0.0.1",
			Port: "80",
		},
		{
			Type: TypeHashIPPort,
			IP:   "10.0.0.2",
			Port: "81",
		},
		{
			Type: TypeHashIPPort,
			IP:   "10.0.0.3",
			Port: "82",
		},
	}

	for i:= 0; i<3; i++{
		err := ipset.AddEntry(testIPSet[2],testEntry3[i],true)
		if err!= nil{
			t.Errorf("add entry %s,%s to ipset %s,%s error: %v", testEntry3[i].Port, testEntry3[i].Port, testIPSet[2].Name, testIPSet[2].Type, err)
		}
	}

	err = ipset.CreateIPSet(testIPSet[3],true)
	if err != nil{
		t.Errorf("create ipset fail name:%s type:%s err:%v", testIPSet[3].Name, testIPSet[3].Type, err)
	}

	testEntry4 := []*Entry{
		{
			Type: TypeHashNetPort,
			Net:  "10.0.0.0/24",
			Port: "80",
		},
		{
			Type: TypeHashNetPort,
			Net:  "10.0.1.0/24",
			Port: "81",
		},
		{
			Type: TypeHashNetPort,
			Net:  "10.0.2.0/24",
			Port: "82",
		},
	}

	for i:= 0; i<3; i++{
		err := ipset.AddEntry(testIPSet[3],testEntry4[i],true)
		if err!= nil{
			t.Errorf("add entry %s,%s to ipset %s,%s error: %v", testEntry4[i].Net, testEntry4[i].Port, testIPSet[3].Name, testIPSet[3].Type, err)
		}
	}
}

func TestEnnIPSet_DelEntry(t *testing.T) {

	execer := exec.New()
	ipset := NewEnnIPSet(execer)


	testEntry := []*Entry{
		{
			Type: TypeHashIP,
			IP:   "10.0.0.1",
		},
		{
			Type: TypeHashIP,
			IP:   "10.0.0.2",
		},
		{
			Type: TypeHashNet,
			Net:  "10.0.2.0/24",
		},
		{
			Type: TypeHashIPPort,
			IP:   "10.0.0.3",
			Port: "82",
		},
		{
			Type: TypeHashNetPort,
			Net:  "10.0.2.0/24",
			Port: "82",
		},
	}

	err := ipset.DelEntry(testIPSet[0], testEntry[0], false)
	if err != nil{
		t.Errorf("delete entry %s from ipset %s:%s fail: %s", testEntry[0].IP, testIPSet[0].Name, testIPSet[0].Type, err)
	}
	err = ipset.DelEntry(testIPSet[0], testEntry[1], false)
	if err != nil{
		t.Errorf("delete entry %s from ipset %s:%s fail: %s", testEntry[1].IP, testIPSet[0].Name, testIPSet[0].Type, err)
	}
	err = ipset.DelEntry(testIPSet[1], testEntry[2], false)
	if err != nil{
		t.Errorf("delete entry %s from ipset %s:%s fail: %s", testEntry[2].Net, testIPSet[1].Name, testIPSet[1].Type, err)
	}
	err = ipset.DelEntry(testIPSet[2], testEntry[3], false)
	if err != nil{
		t.Errorf("delete entry %s,%s from ipset %s:%s fail: %s", testEntry[3].IP, testEntry[3].Port, testIPSet[2].Name, testIPSet[2].Type, err)
	}
	err = ipset.DelEntry(testIPSet[3], testEntry[4], false)
	if err != nil{
		t.Errorf("delete entry %s,%s from ipset %s:%s fail: %s", testEntry[4].Net, testEntry[4].Port, testIPSet[3].Name, testIPSet[3].Type, err)
	}

}

func TestEnnIPSet_ListEntry(t *testing.T) {

	execer := exec.New()
	ipset := NewEnnIPSet(execer)


	testEntry1 := []*Entry{
		{
			Type: TypeHashIP,
			IP:   "10.0.0.3",
		},
	}

	testEntry2 := []*Entry{
		{
			Type: TypeHashNet,
			Net:  "10.0.0.0/24",
		},
		{
			Type: TypeHashNet,
			Net:  "10.0.1.0/24",
		},
	}

	testEntry3 := []*Entry{
		{
			Type: TypeHashIPPort,
			IP:   "10.0.0.1",
			Port: "80",
		},
		{
			Type: TypeHashIPPort,
			IP:   "10.0.0.2",
			Port: "81",
		},
	}

	testEntry4 := []*Entry{
		{
			Type: TypeHashNetPort,
			Net:  "10.0.0.0/24",
			Port: "80",
		},
		{
			Type: TypeHashNetPort,
			Net:  "10.0.1.0/24",
			Port: "81",
		},
	}

	kernelEntry, err := ipset.ListEntry(testIPSet[0])
	if err != nil{
		t.Errorf("list entry for set %s:%s error", testIPSet[0].Name, testIPSet[0].Type)
	} else {
		isCorrectEntry(t, testEntry1, kernelEntry, testIPSet[0].Type)
	}

	kernelEntry, err = ipset.ListEntry(testIPSet[1])
	if err != nil{
		t.Errorf("list entry for set %s:%s error", testIPSet[1].Name, testIPSet[1].Type)
	} else {
		isCorrectEntry(t, testEntry2, kernelEntry, testIPSet[1].Type)
	}

	kernelEntry, err = ipset.ListEntry(testIPSet[2])
	if err != nil{
		t.Errorf("list entry for set %s:%s error", testIPSet[2].Name, testIPSet[2].Type)
	} else {
		isCorrectEntry(t, testEntry3, kernelEntry, testIPSet[2].Type)
	}

	kernelEntry, err = ipset.ListEntry(testIPSet[3])
	if err != nil{
		t.Errorf("list entry for set %s:%s error", testIPSet[3].Name, testIPSet[3].Type)
	} else {
		isCorrectEntry(t, testEntry4, kernelEntry, testIPSet[3].Type)
	}

}

func TestEnnIPSet_FlushIPSet(t *testing.T) {

	execer := exec.New()
	ipset := NewEnnIPSet(execer)


	for i := 0; i<4; i++ {
		err := ipset.FlushIPSet(testIPSet[i])
		if err != nil{
			t.Errorf("flush ipset %s:%s err %v", testIPSet[i].Name, testIPSet[i].Type, err)
		} else {
			entry, _ := ipset.ListEntry(testIPSet[i])
			if len(entry) != 0{
				t.Errorf("ipset %s:%s entry is %d expect 0", testIPSet[i].Name, testIPSet[i].Type, len(entry))
			}
		}
	}

	for i:= 0; i<4; i++{
		ipset.DestroyIPSet(testIPSet[i])
	}
}

func isCorrectSet(t *testing.T, testIPSet *IPSet, kernelSet *IPSet){

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

func isFindSetName(name string, ipsets []string) bool{

	for _, ipset := range ipsets{
		if name == ipset{
			return true
		}
	}
	return false
}

func isCorrectEntry(t *testing.T, testEntry []*Entry, kernelEntry []*Entry, entryType string){

	if len(testEntry) != len(kernelEntry){
		t.Errorf("entry number is not correct %d, expect %d", len(kernelEntry), len(testEntry))
		return
	}

	for _, entry := range testEntry{
		find := false
		for _, k_entry := range kernelEntry{
			if entry.Type != entryType{
				t.Errorf("invalid entryType %s, expected %s", entryType, entry.Type)
			}
			switch  entryType{
			case TypeHashIP:
				if entry.IP == k_entry.IP{
					find = true
					break
				}
			case TypeHashNet:
				if entry.Net == k_entry.Net{
					find = true
					break
				}
			case TypeHashIPPort:
				if entry.IP == k_entry.IP && entry.Port == k_entry.Port{
					find = true
					break
				}
			case TypeHashNetPort:
				if entry.Net == k_entry.Net && entry.Port == k_entry.Port{
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