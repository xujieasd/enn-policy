## User's Guide

### environment request

* kubernetes version: >=1.7
* kube-proxy: iptables
* network: flannel
* flannel enable SNAT (-ip-masq=true), docker disable SNAT (-ip-masq=false)

### build binary

* git clone https://github.com/xujieasd/enn-policy.git
* make

### build docker image

* git clone https://github.com/xujieasd/enn-policy.git
* docker build -t xxx/enn-policy .

### commend line options
- _commend line usage_

```
$ ./enn-policy --help
Usage of ./enn-policy:
      --cleanup-config                If true cleanup all ipset/iptables rules and exit.
      --config-sync-period duration   How often configuration from the apiserver is refreshed.  Must be greater than 0. (default 15m0s)
      --hostname-override string      If non-empty, will use this string as identification instead of the actual hostname.
      --ip-range string               the ip-range will restrict the policy range, enn-policy is only effective within the ip-range (default value is 0.0.0.0/0) (default "0.0.0.0/0")
      --kubeconfig string             Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --log-dir string                If none empty, write log files in this directory
      --logtostderr                   If true will log to standard error instead of files
      --master string                 The address of the Kubernetes API server (overrides any value in kubeconfig)
      --min-sync-period duration      The minimum interval of how often the iptables rules can be refreshed as endpoints and services change (e.g. '5s', '1m', '2h22m').
      --sync-period duration          The maximum interval of how often ipvs rules are refreshed (e.g. '5s', '1m', '2h22m').  Must be greater than 0. (default 15m0s)
      --v string                      Log level for V logs
      --version                       If true will show enn-policy version number.
```
- _commend line requirement_

```
-- hostname-override           : required
-- kubeconfig                  : required if mater not set
-- master                      : required if kubeconfig not set
-- config-sync-period duration : default by 15m if not set
-- sync-period                 : default by 15m if not set
-- min-sync-period             : default by 0 if not set
-- ip-range                    : default by 0.0.0.0/0 if not set
-- log-dir                     : required if logtostderr is false
-- v                           : required if logtostderr is false
```

### run enn-policy

- _run with basic commend_

```
$ sudo ./enn-policy --kubeconfig /etc/kubernetes/kubeconfig --hostname-override 1192.168.1.10
```

- _project log to file_

```
$ sudo mkdir /var/log/enn-policy
$ sudo ./enn-policy --kubeconfig /etc/kubernetes/kubeconfig --hostname-override 1192.168.1.10 --logtostderr=false --log-dir=/var/log/enn-policy --v=4
```

- _restrict the policy within flannel ip range_

```
$ ip route
default via 10.19.138.1 dev enp0s3 
10.19.138.0/24 dev enp0s3  proto kernel  scope link  src 10.19.138.91 
10.244.0.0/16 dev flannel.1 <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
10.244.1.0/24 dev cni0  proto kernel  scope link  src 10.244.1.1 
169.254.0.0/16 dev enp0s3  scope link  metric 1000 
172.17.0.0/16 dev docker0  proto kernel  scope link  src 172.17.0.1 linkdown
$ sudo ./enn-policy --kubeconfig /etc/kubernetes/kubeconfig --hostname-override 1192.168.1.10 --ip-range=10.244.0.0/16
```

### run as daemenset

- [enn-policy-ds.yaml](../install/daemonset/enn-policy-ds.yaml)

```
$ kubectl create -f enn-policy-ds.yaml
```

### run as systemd service

- please follow the steps to deploy enn-policy as [systemd service](../install/systemd/README.md)

### clean up

- clean up all iptables rules and ipsets created by enn-policy

```
$ sudo ./enn-policy --clenup-config
```