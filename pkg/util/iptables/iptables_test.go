package iptables

import (

	"testing"
	"strings"
)

const (
	FILTER_TABLE       = "filter"
	INPUT_CHAIN        = "INPUT"
	OUTPUT_CHAIN       = "OUTPUT"
	FORWARD_CHAIN      = "FORWARD"
	TESTIN_CHAIN       = "TEST-INPUT"
	TESTOU_CHAIN       = "TEST-OUTPUT"
	TESTFO_CHAIN       = "TEST-FORWARD"
)

var ipt = NewEnnIPTables()

func TestEnnIPTables_PrependUnique(t *testing.T) {

	err := ipt.NewChain(FILTER_TABLE, TESTFO_CHAIN)
	if err != nil{
		t.Errorf("create new chain err: %v", err)
		return
	}

	var args []string

	args = []string{
		"-j", TESTFO_CHAIN,
	}

	err = ipt.PrependUnique(FILTER_TABLE, FORWARD_CHAIN, args...)
	if err!= nil{
		t.Errorf("prepend chain err: %v", err)
		return
	}

	ok := checkPrepend(t)
	if !ok{
		t.Errorf("first time check prepend unique fail")
		return
	}

	args1 := []string{
		"-m", "comment", "--comment", "aaa",
	}
	args2 := []string{
		"-m", "comment", "--comment", "bbb",
	}
	ipt.PrependUnique(FILTER_TABLE, FORWARD_CHAIN, args1...)
	ipt.PrependUnique(FILTER_TABLE, FORWARD_CHAIN, args2...)

	// prepend again and ensure this is always the first rule
	err = ipt.PrependUnique(FILTER_TABLE, FORWARD_CHAIN, args...)
	if err!= nil{
		t.Errorf("prepend chain err: %v", err)
		return
	}

	ok = checkPrepend(t)
	if !ok{
		t.Errorf("second time check prepend unique fail")
		return
	}

	// clean up
	ipt.Delete(FILTER_TABLE, FORWARD_CHAIN, args...)
	ipt.Delete(FILTER_TABLE, FORWARD_CHAIN, args1...)
	ipt.Delete(FILTER_TABLE, FORWARD_CHAIN, args2...)
	ipt.DeleteChain(FILTER_TABLE, TESTFO_CHAIN)
}

func checkPrepend(t *testing.T) bool{

	lists, err := ipt.List(FILTER_TABLE, FORWARD_CHAIN)
	if err!= nil{
		t.Errorf("list chain err: %v", err)
		return false
	}

	if !strings.Contains(lists[1], TESTFO_CHAIN){
		t.Errorf("get unexpected chain, shoud contain: %s", TESTFO_CHAIN)
		t.Errorf("chain is: %s", lists[1])
		return false
	}

	count := 0
	for i, list := range lists{
		if strings.Contains(list, TESTFO_CHAIN){
			count ++
		}
		if count > 1{
			t.Errorf("get more than 1 chain contains: %s", TESTFO_CHAIN)
			t.Errorf("chain number: %d", i)
			return false
		}
	}
	return true
}