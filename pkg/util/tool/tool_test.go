package tool

import (
	"testing"
	"strings"
	"fmt"
)

func TestPowerInt(t *testing.T) {

	var result int
	result = PowerInt(2, 0)

	if result != 1{
		t.Errorf("power result is %d, expect %d", result, 1)
	}

	result = PowerInt(2, 10)

	if result != 1024{
		t.Errorf("power result is %d, expect %d", result, 1024)
	}

	result = PowerInt(-2, 9)

	if result != -512{
		t.Errorf("power result is %d, expect %d", result, -512)
	}

	result = PowerInt(0, 0)

	if result != 1{
		t.Errorf("power result is %d, expect %d", result, 1)
	}

	result = PowerInt(0, 100)

	if result != 0{
		t.Errorf("power result is %d, expect %d", result, 0)
	}

}

func TestIpStringToInt(t *testing.T) {

	ip := "10.172.16.0"
	ips := strings.Split(ip, ".")
	ipsInt, _ := IpStringToInt(ips)
	err := checkIpInt(t, 10, 172, 16, 0, ipsInt)
	if err!= nil{
		t.Errorf("check result error: %v", err)
	}

	ip = "255.255.255.255"
	ips = strings.Split(ip, ".")
	ipsInt, _ = IpStringToInt(ips)
	err = checkIpInt(t, 255, 255, 255, 255, ipsInt)
	if err!= nil{
		t.Errorf("check result error: %v", err)
	}

	ip = "0.0.0.0"
	ips = strings.Split(ip, ".")
	ipsInt, _ = IpStringToInt(ips)
	err = checkIpInt(t, 0, 0, 0, 0, ipsInt)
	if err!= nil{
		t.Errorf("check result error: %v", err)
	}
}

func TestIpIntToString(t *testing.T) {

	var intIPs []int
	intIPs = make([]int, 4)

	intIPs[3] = 10
	intIPs[2] = 172
	intIPs[1] = 16
	intIPs[0] = 0

	ip, _ := IpIntToString(intIPs)

	if strings.Compare(ip, "10.172.16.0") != 0 {
		t.Errorf("wrong ip string %s, expect %s", ip, "10.172.16.0")
	}

	intIPs[3] = 255
	intIPs[2] = 255
	intIPs[1] = 255
	intIPs[0] = 255

	ip, _ = IpIntToString(intIPs)

	if strings.Compare(ip, "255.255.255.255") != 0 {
		t.Errorf("wrong ip string %s, expect %s", ip, "255.255.255.255")
	}

	intIPs[3] = 0
	intIPs[2] = 0
	intIPs[1] = 0
	intIPs[0] = 0

	ip, _ = IpIntToString(intIPs)

	if strings.Compare(ip, "0.0.0.0") != 0 {
		t.Errorf("wrong ip string %s, expect %s", ip, "0.0.0.0")
	}
}

func TestIpOperateAdd(t *testing.T) {

	var intIPs []int
	intIPs = make([]int, 4)
	var baseItem int
	var step int

	intIPs[3] = 172
	intIPs[2] = 16
	intIPs[1] = 0
	intIPs[0] = 0

	baseItem = 1
	step = 16

	ips, _ := IpOperateAdd(intIPs, baseItem, step)
	err := checkIpInt(t, 172, 16, 16, 0, ips)
	if err!= nil{
		t.Errorf("check result error: %v", err)
	}

	intIPs[3] = 172
	intIPs[2] = 16
	intIPs[1] = 0
	intIPs[0] = 0

	baseItem = 0
	step = 1

	ips, _ = IpOperateAdd(intIPs, baseItem, step)
	err = checkIpInt(t, 172, 16, 0, 1, ips)
	if err!= nil{
		t.Errorf("check result error: %v", err)
	}

	intIPs[3] = 172
	intIPs[2] = 16
	intIPs[1] = 240
	intIPs[0] = 0

	baseItem = 1
	step = 16

	ips, _ = IpOperateAdd(intIPs, baseItem, step)
	err = checkIpInt(t, 172, 17, 0, 0, ips)
	if err!= nil{
		t.Errorf("check result error: %v", err)
	}

	intIPs[3] = 172
	intIPs[2] = 255
	intIPs[1] = 255
	intIPs[0] = 254

	baseItem = 0
	step = 3

	ips, _ = IpOperateAdd(intIPs, baseItem, step)
	err = checkIpInt(t, 173, 0, 0, 1, ips)
	if err!= nil{
		t.Errorf("check result error: %v", err)
	}
}

func checkIpInt(t *testing.T, a3, a2, a1, a0 int, ips []int) error{

	if len(ips) != 4{
		t.Errorf("invalid lengh: %d, expect %d", len(ips), 4)
		return fmt.Errorf("invalid lengh")
	}

	if a3 == ips[3] && a2 == ips[2] && a1 == ips[1] && a0 == ips[0]{
		return nil
	}

	t.Errorf("result %d.%d.%d.%d, expect %d.%d.%d.%d", ips[3], ips[2], ips[1], ips[0], a3, a2, a1, a0)
	return fmt.Errorf("wrong result")
}