## Developer's Guide

### Environment

* kubernetes version: >=1.7
* kube-proxy: iptables
* network: flannel
* flannel enable SNAT (-ip-masq=true), docker disable SNAT (-ip-masq=false)
* build environment: golang

### How to build

* git clone https://github.com/xujieasd/enn-policy.git
* make

### How to build image

* docker build -t xxx/enn-policy .

### How to Run test case

- [Unit Test](../test/unit_test/ReadMe.md)
- [Stress Test](../test/pressure_test/README.md)