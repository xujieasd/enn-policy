package iptables

import (
	"github.com/coreos/go-iptables/iptables"
	"github.com/golang/glog"
	"fmt"
)

type EnnIPTables struct {
	IPTablesHandle    *iptables.IPTables
}

type Interface interface {
	// Exists will check whether given rulespec in specified table/chain exists
	Exists(table, chain string, rulespec ...string) (bool, error)
	// Prepend inserts rulespec to specified table/chain (in specified pos)
	Prepend(table, chain string, pos int, rulespec ...string) error
	// PrependUnique acts like Prepend except that it won't add a duplicate
	PrependUnique(table, chain string, rulespec ...string) error
	// Append appends rulespec to specified table/chain
	Append(table, chain string, rulespec ...string) error
	// AppendUnique acts like Append except that it won't add a duplicate
	AppendUnique(table, chain string, rulespec ...string) error
	// Delete removes rulespec in specified table/chain
	Delete(table, chain string, rulespec ...string) error
	// List rules in specified table/chain
	List(table, chain string) ([]string, error)
	// ListChains returns a slice containing the name of each chain in the specified table.
	ListChains(table string) ([]string, error)
	// NewChain creates a new chain in the specified table.
	NewChain(table, chain string) error
	// ClearChain flushed (deletes all rules) in the specified table/chain. If the chain does not exist, a new one will be created
	ClearChain(table, chain string) error
	// DeleteChain deletes the chain in the specified table. The chain must be empty
	DeleteChain(table, chain string) error
}

func NewEnnIPTables() Interface{

	handle, err := iptables.New()
	if err != nil {
		glog.Errorf("InitIPTablesInterface failed Error: %v", err)
		panic(err)
	}

	var e = &EnnIPTables{
		IPTablesHandle: handle,
	}

	return e
}

// Exists will check whether given rulespec in specified table/chain exists
func (e *EnnIPTables) Exists(table, chain string, rulespec ...string) (bool, error) {
	exists, err := e.IPTablesHandle.Exists(table, chain, rulespec...)
	return exists, err
}

// Prepend inserts rulespec to specified table/chain (in specified pos)
func (e *EnnIPTables) Prepend(table, chain string, pos int, rulespec ...string) error {
	err := e.IPTablesHandle.Insert(table, chain, pos, rulespec...)
	if err != nil{
		return fmt.Errorf("iptables insert chain %s to tables %s err %s", chain, table, err.Error())
	}
	return nil
}

// PrependUnique acts like Prepend except that it won't add a duplicate
func (e *EnnIPTables) PrependUnique(table, chain string, rulespec ...string) error {
	exists, err := e.IPTablesHandle.Exists(table, chain, rulespec...)
	if err != nil {
		return err
	}
	if !exists {
		err := e.IPTablesHandle.Insert(table, chain, 1, rulespec...)
		if err != nil{
			return fmt.Errorf("iptables prependUnique chain %s to tables %s err %s", chain, table, err.Error())
		}
	}
	return nil
}

// Append appends rulespec to specified table/chain
func (e *EnnIPTables) Append(table, chain string, rulespec ...string) error {
	err := e.IPTablesHandle.Append(table, chain, rulespec...)
	if err != nil{
		return fmt.Errorf("iptables append chain %s to tables %s err %s", chain, table, err.Error())
	}
	return nil
}

// AppendUnique acts like Append except that it won't add a duplicate
func (e *EnnIPTables) AppendUnique(table, chain string, rulespec ...string) error {
	err := e.IPTablesHandle.AppendUnique(table, chain, rulespec...)
	if err != nil{
		return fmt.Errorf("iptables appendUnique chain %s to tables %s err %s", chain, table, err.Error())
	}
	return nil
}

// Delete removes rulespec in specified table/chain
func (e *EnnIPTables) Delete(table, chain string, rulespec ...string) error {
	err := e.IPTablesHandle.Delete(table, chain, rulespec...)
	if err != nil{
		return fmt.Errorf("iptables delete chain %s to tables %s err %s", chain, table, err.Error())
	}
	return nil
}

// List rules in specified table/chain
func (e *EnnIPTables) List(table, chain string) ([]string, error) {
	list, err := e.IPTablesHandle.List(table, chain)
	if err != nil{
		return nil, fmt.Errorf("iptables list chain %s to tables %s err %s", chain, table, err.Error())
	}
	return list, nil
}

// ListChains returns a slice containing the name of each chain in the specified table.
func (e *EnnIPTables) ListChains(table string) ([]string, error) {
	list, err := e.IPTablesHandle.ListChains(table)
	if err != nil{
		return nil, fmt.Errorf("iptables list tables %s err %s", table, err.Error())
	}
	return list, nil
}

// NewChain creates a new chain in the specified table.
// iptables -t [table] -N [chainName]
func (e *EnnIPTables) NewChain(table, chain string) error {
	err := e.IPTablesHandle.NewChain(table, chain)
	// If the chain already exists, it will result in an error.
	if err != nil && err.(*iptables.Error).ExitStatus() != 1{
		return fmt.Errorf("new iptables chain %s to tables %s err %s", chain, table, err.Error())
	}
	return nil
}

// ClearChain flushed (deletes all rules) in the specified table/chain.
// If the chain does not exist, a new one will be created
// iptables -t [table] -F [chainName]
func (e *EnnIPTables) ClearChain(table, chain string) error {
	err := e.IPTablesHandle.ClearChain(table, chain)
	if err != nil{
		return fmt.Errorf("iptables flush chain %s to tables %s err %s", chain, table, err.Error())
	}
	return nil
}

// DeleteChain deletes the chain in the specified table. The chain must be empty
// iptables -t [table] -X [chainName]
func (e *EnnIPTables) DeleteChain(table, chain string) error {
	err := e.IPTablesHandle.DeleteChain(table, chain)
	if err != nil{
		return fmt.Errorf("iptables delete chain %s to tables %s err %s", chain, table, err.Error())
	}
	return nil
}