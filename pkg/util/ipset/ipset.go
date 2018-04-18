package ipset

import (
	"fmt"
	"strings"
	"strconv"

	"k8s.io/utils/exec"
	"github.com/golang/glog"
)

const IPSetCmd = "ipset"

const(
	TypeHashIP       = "hash:ip"
	TypeHashNet      = "hash:net"
	TypeHashIPPort   = "hash:ip,port"
	TypeHashNetPort  = "hash:net,port"
)

const(
	DefaultMaxElem  = 65536
	DefaultHashSize = 1024
)

const(
	FamilyIPV4  = "inet"
	FamilyIPV6  = "inet6"
)

const(
	ProtocolTCP = "tcp"
	ProtocolUDP = "udp"
)

//    example ipset structure by typing ipset list
//    Name: subip
//    Type: hash:net
//    Revision: 6
//    Header: family inet hashsize 1024 maxelem 65536
//    Size in memory: 448
//    References: 1
//    Members:
//    10.244.0.0/16

type IPSet struct {
	Name        string
	Type        string
	Family      string
	HashSize    int
	MaxElem     int
	Reference   int

}

type Entry struct {
	Type        string
	IP          string
	Port        string
	Net         string
}

type EnnIPSet struct {
	exec exec.Interface
}

type Interface interface {
	// FlushALl flash all ipsets in system
	FlushAll() error
	// DestroyAll destroy all ipsets in system
	DestroyAll() error
	// FlushIPSet flush all entries from the specified set
	FlushIPSet(set *IPSet) error
	// CreateIPSet create new ipset, if ignoreExitErr is true, ipset ignores the error otherwise raised when the some set already exists
	CreateIPSet(set *IPSet, ignoreExitErr bool) error
	// DestoryIPSet destroy the specified set
	DestroyIPSet(set *IPSet) error
	// ListIPSetsName list all the ipset names created by enn-policy
	ListIPSetsName() ([]string, error)
	// GetIPSet list the header data for a named ipset
	GetIPSet(name string) (*IPSet, error)
	// AddEntry add a given entry to the set, if ignoreExitErr is true, ipset ignores if the entry already added to the set
	AddEntry(set *IPSet, entry *Entry, ignoreExitErr bool) error
	// DelEntry delete an entry from a set, if ignoreExitErr is true, ipset ignores if the entry is not in the set
	DelEntry(set *IPSet, entry *Entry, ignoreExitErr bool) error
	// ListEntry list all the entries for the specified set
	ListEntry(set *IPSet) ([]*Entry, error)
}

func NewEnnIPSet(exec exec.Interface) Interface{
	return &EnnIPSet{
		exec:    exec,
	}
}

func (e *EnnIPSet) FlushAll() error{

	args := []string{"flush"}
	_, err := e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return fmt.Errorf("Flush all ipset fail: %v", err)
	}

	glog.V(6).Infof("successful flush all ipset")
	return nil
}

func (e *EnnIPSet) DestroyAll() error{

	args := []string{"destroy"}
	_, err := e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return fmt.Errorf("destroy all ipset fail: %v", err)
	}

	glog.V(6).Infof("successful destroy all ipset")
	return nil
}

func (e *EnnIPSet) FlushIPSet(set *IPSet) error{

	args := []string{"flush", set.Name}
	_, err := e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return fmt.Errorf("Flush ip set %s/%s fail: %v", set.Name, set.Type, err)
	}

	glog.V(6).Infof("successful flush ipset %s", set.Name)
	return nil
}

func (e *EnnIPSet) CreateIPSet(set *IPSet, ignoreExitErr bool) error{

	args := []string{"create", set.Name, set.Type}
	if ignoreExitErr{
		args = append(args, "-exist")
	}
	_, err := e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return fmt.Errorf("create ip set %s/%s fail: %v", set.Name, set.Type, err)
	}

	glog.V(6).Infof("successful create ipset %s:%s", set.Name, set.Type)
	return nil
}

func (e *EnnIPSet) DestroyIPSet(set *IPSet) error{

	args := []string{"destroy", set.Name}
	_, err := e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return fmt.Errorf("destory ip set %s/%s fail: %v", set.Name, set.Type, err)
	}

	glog.V(6).Infof("successful destory ipset %s", set.Name)
	return nil
}

func (e *EnnIPSet) ListIPSetsName() ([]string, error){

	args := []string{"list", "-n"}
	out, err := e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return nil, fmt.Errorf("list ipset names fail: %v", err)
	}
	setName := strings.Split(string(out),"\n")

	// if string spilt by "\n", should notice that the last element could be ""
	// e.g
	// ipset list -n
	// test1\n
	// test2\n
	// so after split, the result is {"test1","test2",""}
	if setName[len(setName)-1] == ""{
		setName = setName[:len(setName)-1]
	}

	return setName, nil
}

func (e *EnnIPSet) GetIPSet(name string) (*IPSet, error){

	args := []string{"list", name}
	out, err := e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return nil, fmt.Errorf("list ipset %s fail: %v", name, err)
	}

	strs := strings.Split(string(out),"\n")

	ipSet := &IPSet{}

	for _, str := range strs{
		if strings.HasPrefix(str, "Name"){
			str_name := strings.Split(str,": ")
			ipSet.Name = str_name[1]
		}else if strings.HasPrefix(str, "Type"){
			str_type := strings.Split(str,": ")
			ipSet.Type = str_type[1]
		}else if strings.HasPrefix(str, "Header"){
			str_head := strings.Split(str,": ")
			set_head := str_head[1]
			str_head_info := strings.Split(set_head, " ")
			ipSet.Family   = str_head_info[1]
			ipSet.HashSize, err = strconv.Atoi(str_head_info[3])
			if err!= nil{
				glog.Errorf("get hashsize of ipset %s fail %v", name, err)
			}
			ipSet.MaxElem, err = strconv.Atoi(str_head_info[5])
			if err!= nil{
				glog.Errorf("get maxelem of ipset %s fail %v", name, err)
			}
		}else if strings.HasPrefix(str, "Reference"){
			str_reference := strings.Split(str,": ")
			ipSet.Reference, _ = strconv.Atoi(str_reference[1])
		}else if strings.HasPrefix(str, "Members"){
			break
		}
	}

	return ipSet, nil
}

func (e *EnnIPSet) AddEntry(set *IPSet, entry *Entry, ignoreExitErr bool) error{

	entry_string, err := EntryToString(entry)

	if err!= nil{
		return fmt.Errorf("addEntry fail %v", err)
	}
	args := []string{"add", set.Name, entry_string}
	if ignoreExitErr{
		args = append(args, "-exist")
	}
	_, err = e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return fmt.Errorf("add entry %s to ip set %s/%s fail: %v", entry_string, set.Name, set.Type, err)
	}

	glog.V(6).Infof("successful add entry %s to ipset %s:%s", entry_string, set.Name, set.Type)
	return nil
}

func (e *EnnIPSet) DelEntry(set *IPSet, entry *Entry, ignoreExitErr bool) error{

	entry_string, err := EntryToString(entry)
	if err!= nil{
		return fmt.Errorf("addEntry fail %v", err)
	}
	args := []string{"del", set.Name, entry_string}
	if ignoreExitErr{
		args = append(args, "-exist")
	}
	_, err = e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return fmt.Errorf("delete entry %s to ip set %s/%s fail: %v", entry_string, set.Name, set.Type, err)
	}

	glog.V(6).Infof("successful del entry %s from ipset %s:%s", entry_string, set.Name, set.Type)
	return nil
}

func (e *EnnIPSet) ListEntry(set *IPSet) ([]*Entry, error){

	args := []string{"list", set.Name}
	out, err := e.exec.Command(IPSetCmd, args...).CombinedOutput()
	if err != nil{
		return nil, fmt.Errorf("list ipset entry %s/%s fail: %v", set.Name, set.Type, err)
	}

	entries := make([]*Entry, 0)
	strs := strings.Split(string(out),"\n")
	isEntry := false
	for _, str := range strs {
		if isEntry{
			if str == ""{
				// if string spilt by "\n", should notice that the last element could be ""
				continue
			}
			entry, err := StringToEntry(set.Type, str)
			if err!= nil{
				glog.Errorf("list entry fail because invalid entry %s : %v", str, err)
				continue
			}
			entries = append(entries, entry)
		}else{
			if strings.HasPrefix(str, "Members"){
				isEntry = true
			}
		}
	}

	return entries, nil
}

func EntryToString(entry *Entry) (string, error){

	switch entry.Type {
	case TypeHashIP:
		return fmt.Sprintf("%s",entry.IP), nil
	case TypeHashNet:
		return fmt.Sprintf("%s",entry.Net), nil
	case TypeHashIPPort:
		return fmt.Sprintf("%s,%s",entry.IP,entry.Port), nil
	case TypeHashNetPort:
		return fmt.Sprintf("%s,%s",entry.Net,entry.Port), nil
	}

	return "", fmt.Errorf("invalid entry type")
}

func StringToEntry(entryType string, entryElem string) (*Entry, error){

	entry := &Entry{
		Type:  entryType,
	}

	switch entryType {
	case TypeHashIP:
		entry.IP = entryElem
	case TypeHashNet:
		entry.Net = entryElem
	case TypeHashIPPort:
		str := strings.Split(entryElem, ",")
		if len(str) != 2{
			return nil, fmt.Errorf("invalid entry type:%s, member:%s", entryType, entryElem)
		}
		entry.IP   = str[0]
		str = strings.Split(str[1], ":")
		if len(str) != 2{
			return nil, fmt.Errorf("invalid entry type:%s, member:%s", entryType, entryElem)
		}
		entry.Port = str[1]
	case TypeHashNetPort:
		str := strings.Split(entryElem, ",")
		if len(str) != 2{
			return nil, fmt.Errorf("invalid entry type:%s, member:%s", entryType, entryElem)
		}
		entry.Net  = str[0]
		str = strings.Split(str[1], ":")
		if len(str) != 2{
			return nil, fmt.Errorf("invalid entry type:%s, member:%s", entryType, entryElem)
		}
		entry.Port = str[1]
	default:
		return nil, fmt.Errorf("invalid entry type")
	}

	return entry, nil
}