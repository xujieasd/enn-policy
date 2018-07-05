package tool

import (
	"fmt"
	"strconv"
)

// support n >= 0
func PowerInt(x, n int) int{
	result := 1
	if n!=0{
		for i := 0; i < n; i++{
			result = result * x
		}
	}
	return result
}

func IpStringToInt(ips []string) ([]int, error){
	if len(ips) != 4{
		return nil, fmt.Errorf("len of ips:%d is not 4", len(ips))
	}

	var intIPs []int
	intIPs = make([]int, 4)
	for i := 0; i < 4; i++{
		ip, err := strconv.Atoi(ips[i])
		if err != nil{
			return nil, fmt.Errorf("unexpected ip:index : %s:%d, err:%v", ips[i], i, err)
		}
		if ip > 255 || ip < 0 {
			return nil, fmt.Errorf("ip:index : %s:%d out of range", ips[i], i)
		}
		intIPs[3 - i] = ip

	}
	return intIPs, nil
}

func IpIntToString(intIPs []int) (string, error){

	if len(intIPs) != 4{
		return "", fmt.Errorf("len of ips:%d is not 4", len(intIPs))
	}
	result := fmt.Sprintf("%s.%s.%s.%s",
		strconv.Itoa(intIPs[3]),
		strconv.Itoa(intIPs[2]),
		strconv.Itoa(intIPs[1]),
		strconv.Itoa(intIPs[0]),
	)
	return result, nil
}

func IpOperateAdd(intIPs []int, baseItem int, step int) ([]int, error){

	if len(intIPs) != 4{
		return intIPs, fmt.Errorf("len of ips:%d is not 4", len(intIPs))
	}

	var result []int
	result = make([]int, 4)
	for i := 0 ; i < 4; i++{
		result[i] = intIPs[i]
	}

	item := baseItem
	// step should be less than 256
	carry := step
	for item < 4 {
		result[item] = result[item] + carry
		if result[item] >= 256{
			result[item] = result[item] - 256
			carry = 1
		} else {
			carry = 0
			break
		}
		item ++
	}
	if carry != 0{
		return intIPs, fmt.Errorf("invalid ips: %d.%d.%d.%d", intIPs[3], intIPs[2], intIPs[1], intIPs[0])
	}

	return result, nil

}
