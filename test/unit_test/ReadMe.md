### Unit test environment
* golang version >= 1.6
* iptables
* ipset
* kernel mode access

### How to run Unit test
 * cd to root folder of the project
 * sudo make test
 
 if UT passed, will see result like below
 ```
 enn-policy unit test Starting.
 hack/test.sh
 ok  	enn-policy/pkg/policy	3.398s
 ok  	enn-policy/pkg/policy/config	0.947s
 ok  	enn-policy/pkg/policy/util	0.041s
 ok  	enn-policy/pkg/util/ipset	0.047s
 enn-policy unit test finished.

 ```