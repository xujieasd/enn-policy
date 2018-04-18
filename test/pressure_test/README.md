## Pressure Test Document
*This document will introduce how to setup a stress test environment and how to run stress test to check performance for enn-policy*

### _Statement_

Most of the stress test code are referred to [k8sclient](https://github.com/zq-david-wang/k8sclient)  
This stress code do some change on k8sclient module to match network policy test needs  
For more code detail, please refer to [k8sclient](https://github.com/zq-david-wang/k8sclient)  


### _Environment needs_

- python
- pip
- kubernetes
- k8sclient

### _Installation_

* [pip](https://pip.pypa.io/en/stable/installing/)

```shell
$ wget https://bootstrap.pypa.io/get-pip.py
$ python get-pip.py
```
* [kubernetes](https://github.com/kubernetes-incubator/client-python/)

```shell
$ pip install kubernetes
```
[official kubernates client python api reference](https://github.com/kubernetes-client/python)

* k8sclient

```shell
$ cd test/pressure_test
$ pip install -e .
```

### _Run stress test_

#### 0. Ensure kubernetes config file
make sure you kubernetes config file are under ~/.kube/ directory
> If you are using python2, you may get error message like "hostname 'xxxx' doesn't match either of 'kubernetes', 'kubernetes.default','kubernetes-mater'"  
> This is happening likely due to running python < 3.5, ip hostnames are not supported.  
> You need to modify config file, open kubernetes config file, you will see something like "server: X.X.X.X:6443", change X.X.X.X to a hostname  
> Don't forget to ensure your hostname is in /etc/hosts file  
> [reference](https://github.com/kelproject/pykube/issues/29) 

#### 1. Example to run network throughput test
enn-policy will add large numbers of rules in iptables filter table, which could slow down network traffic  
* _python network_throughput.py --nodePre=[pre]_ 
* --nodePre (The prefix of nodes, e.g run 'kubectl get node', if you get node name is 'ubuntu-clientX' then value here is 'ubuntu-client')  

```shell
$ kubectl get nodes
NAME             STATUS    AGE       VERSION
ubuntu-client1   Ready     45d       v1.8.0
ubuntu-client2   Ready     45d       v1.8.0
ubuntu-master    Ready     45d       v1.8.0
$ cd test/pressure_test/tests
$ python network_throughput.py --nodePre="ubuntu-client"
```

#### 2. Example to run ingress stress test and check if traffic access is correct
* _python pod_create.py --namespaceNumber=[ns_number] --nodePre=[pre]_

> create [ns_number] namespace, for every namespace, create pod in each node, and create svc for each pod, so if you create n namespaces in m nodes, will totally create n*m pods and services

* _python policy_ingress_create.py --namespaceNumber=[ns_number] --policyNumber=[p_numer]_

> create [p_number] ingress network policies for each namespace, for each namespace, will create a special ingress rule to access itself. So will totally create [ns_number]*([p_number+1]) networkPolicies  
> need to wait for a while after run this code, cause iptables may need some time to create rules

* _python traffic_test.py --namespaceNumber=[ns_number] --policyNumber=[p_numer] --nodePre=[pre]_

> will check network traffic from every pods to every svcs, should allow traffics which defined in network policy and reject other traffics

```shell
$ cd test/pressure_test/tests
$ python pod_create.py --namespaceNumber=10 --nodePre="ubuntu-client"
It took 57s to deploy 10 pod on ubuntu-client1
It took 57s to deploy 10 pod on ubuntu-client2
it took 60.9402770996 s
$ python policy_ingress_create.py --namespaceNumber=10 --policyNumber=6
$ python traffic_test.py --namespaceNumber=10 --policyNumber=6 --nodePre="ubuntu-client"
check connection
no issue found
check unconnection
no issue found
```

#### 3. Example clean up all
clean up all pods, all services, all policies created by stress test

```
$ cd test/pressure_test/tests
$ python cleanup.py
```