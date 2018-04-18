### Advanced Daemon

- this document will show some advanced examples to see how enn-policy works
- before go through blew contents, you must have [deployed enn-policy](./user-guide.md) in your k8s environment
- also, you need to have a brief understanding of [kubernetes network policy](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- you can check [simple daemons](./example-daemon.md)

#### Create advanced ingress daemon

**_1. config new namespace_**

- create 4 namespaces:   
p-demo labeled name=p-demo and app=access;  
p-demo1 labeled name=p-demo1;  
p-demo2 labeled name=p-demo2 and app=access;  
p-demo3 labeled name=p-demo3 and app=access1;  
example yaml file [here](./yaml/namespace-p-demo.yaml)

```
$ kubectl create -f namespace-p-demo.yaml 
namespace "p-demo" created
namespace "p-demo1" created
namespace "p-demo2" created
namespace "p-demo3" created
```

**_2. create pods_**

- create some pods in these namespace, see example yaml file [here](./yaml/nginx-policy-pod.yaml)

```
$ kubectl create -f nginx-policy-pod.yaml 
deployment "nginx" created
deployment "nginx1" created
deployment "nginx2" created
deployment "nginx" created
deployment "nginx" created
deployment "nginx" created
```

- check these pods

> here pod nginx-845466769c-k26sc label is app=nginx  
> pod nginx1-78bf97c65d-vw559 label is app=access  
> pod nginx2-54c7bb866-gzksj label is app=reject

```
$ kubectl get pods -o wide --all-namespaces
NAMESPACE      NAME                                    READY     STATUS    RESTARTS   AGE       IP             NODE
p-demo         nginx-845466769c-k26sc                  1/1       Running   0          4m        10.244.1.127   ubuntu-client1
p-demo         nginx1-78bf97c65d-vw559                 1/1       Running   0          4m        10.244.2.124   ubuntu-client2
p-demo         nginx2-54c7bb866-gzksj                  1/1       Running   0          4m        10.244.1.128   ubuntu-client1
p-demo1        nginx-845466769c-h6wlk                  1/1       Running   0          4m        10.244.2.122   ubuntu-client2
p-demo2        nginx-845466769c-2mvhn                  1/1       Running   0          4m        10.244.1.129   ubuntu-client1
p-demo3        nginx-845466769c-88lmx                  1/1       Running   0          4m        10.244.2.123   ubuntu-client2
...

```

**_3. expose deployments as service_**

- create svc for every deployment in namespace p-demo

```
$ kubectl -n p-demo expose deployment/nginx --port=80 --type=NodePort
service "nginx" exposed
$ kubectl -n p-demo expose deployment/nginx1 --port=80 --type=NodePort
service "nginx1" exposed
$ kubectl -n p-demo expose deployment/nginx2 --port=80 --type=NodePort
service "nginx2" exposed

```

- check deployed svc in namespace p-demo

```
$ kubectl -n p-demo get svc -o wide
NAME      CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE       SELECTOR
nginx     10.103.152.3    <nodes>       80:32322/TCP   2m        app=nginx
nginx1    10.97.31.122    <nodes>       80:30579/TCP   2m        app=access
nginx2    10.103.32.152   <nodes>       80:32578/TCP   2m        app=reject

```

**_4. deploy ingress policy_**

- create new ingress policy, you can find yaml file [here](./yaml/p-demo-ingress.yaml)

> this policy means:  
> allow connection to all pods in namespace "p-demo" from any pod in a namespace with label "name=p-demo1"  
> allow connection to all pods in namespace "p-demo" from any pod in namespace "p-demo" with pod label "app=access"  
> allow connection to all pods in namespace "p-demo" from ip addresses are in ip range 10.19.139.0/24  
> reject connection to all pods in namespace "p-demo" to any other ip address

```yaml
$ cat p-demo-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ingress-demo
  namespace: p-demo
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: access
    - namespaceSelector:
        matchLabels:
          name: p-demo1
    - ipBlock:
        cidr: 10.19.139.0/24

```

- deploy this ingress policy

```
$ kubectl create -f p-demo-ingress.yaml 
networkpolicy "ingress-demo" created
```

**_5. check connection_**

- check podSelector

> as network policy defined, in namespace "p-demo" only pod nginx1-78bf97c65d-vw559(label app=access) can visit pods in namespace "p-demo"

```
$ kubectl -n p-demo exec nginx1-78bf97c65d-vw559 -it /bin/bash 
root@nginx1-78bf97c65d-vw559:/# echo > /dev/tcp/10.103.152.3/80
root@nginx1-78bf97c65d-vw559:/# exit
```

```
$ kubectl -n p-demo exec nginx2-54c7bb866-gzksj -it /bin/bash
root@nginx2-54c7bb866-gzksj:/# echo > /dev/tcp/10.103.152.3/80
bash: connect: Connection refused
bash: /dev/tcp/10.103.152.3/80: Connection refused
root@nginx2-54c7bb866-gzksj:/# exit
```

- check namespaceSelector

> as network policy defined, pods in namespace "p-demo1" can visit pods in namespace "p-demo", but pods in namespace "p-demo2" "p-demo3" cannot

```
$ kubectl -n p-demo1 exec nginx-845466769c-h6wlk -it /bin/bash
root@nginx-845466769c-h6wlk:/# echo > /dev/tcp/10.103.152.3/80
root@nginx-845466769c-h6wlk:/# exit
```

```
kubectl -n p-demo2 exec nginx-845466769c-2mvhn -it /bin/bash
root@nginx-845466769c-2mvhn:/# echo > /dev/tcp/10.103.152.3/80
bash: connect: Connection refused
bash: /dev/tcp/10.103.152.3/80: Connection refused
root@nginx-845466769c-2mvhn:/# exit
```

```
 kubectl -n p-demo3 exec nginx-845466769c-88lmx -it /bin/bash
root@nginx-845466769c-88lmx:/# echo > /dev/tcp/10.103.152.3/80
bash: connect: Connection refused
bash: /dev/tcp/10.103.152.3/80: Connection refused
root@nginx-845466769c-88lmx:/# exit
```

- check ipBlock

> as network policy defined, ip address in range 10.19.139.0/24 can visit pods in namespace "p-demo"  

> log in machine 10.19.138.147, and try to connect nodeport

```
$ ssh root@10.19.138.147
$ echo > /dev/tcp/10.19.138.96/32322
bash: connect: Connection refused
bash: /dev/tcp/10.19.138.96/32322: Connection refused
$ exit
```

> log in machine 10.19.139.159, and try to connect nodeport

```
$ ssh root@10.19.139.159
$ echo > /dev/tcp/10.19.138.96/32322
$ exit
```

**_6. check how iptables and ipset works_**

```
$ iptables -S -t filter | grep ENN
-N ENN-DPATCH-DBBM6RRSD5ESYANP
-N ENN-DPATCH-RCVFHUEC5K3IZT6D
-N ENN-DPATCH-ZDWBQO7N7BDCI4YX
-N ENN-FORWARD
-N ENN-INGRESS-ZC2OD5Z5N6SA7IXZ
-N ENN-INPUT
-N ENN-OUTPUT
-N ENN-PLY-IN-O3FD5RCBJHK4JCRC
-A INPUT -j ENN-INPUT
-A FORWARD -j ENN-FORWARD
-A OUTPUT -j ENN-OUTPUT
-A ENN-DPATCH-DBBM6RRSD5ESYANP -m comment --comment "accept rule selected by policy p-demo/ingress-demo: src namespace match name=p-demo1" -m set --match-set ENN-NSSET-5STP5YUNBEIBJW6L src -j ACCEPT
-A ENN-DPATCH-RCVFHUEC5K3IZT6D -m comment --comment "accept rule selected by policy p-demo/ingress-demo: src pod match app=access" -m set --match-set ENN-PODSET-T2GA36CFVTOK3KTQ src -j ACCEPT
-A ENN-DPATCH-ZDWBQO7N7BDCI4YX -s 10.19.139.0/24 -m comment --comment "accept rule selected by policy p-demo/ingress-demo: -s cidr 10.19.139.0/24" -j ACCEPT
-A ENN-FORWARD -m set --match-set ENN-NS-ZC2OD5Z5N6SA7IXZ dst -m comment --comment "ingress entry for namespace/p-demo" -j ENN-INGRESS-ZC2OD5Z5N6SA7IXZ
-A ENN-INGRESS-ZC2OD5Z5N6SA7IXZ -m comment --comment "iprange is default value 0.0.0.0/0 so derectly jump to iPRangeChain" -j ENN-PLY-IN-O3FD5RCBJHK4JCRC
-A ENN-OUTPUT -m set --match-set ENN-NS-ZC2OD5Z5N6SA7IXZ dst -m comment --comment "ingress entry for namespace/p-demo" -j ENN-INGRESS-ZC2OD5Z5N6SA7IXZ
-A ENN-PLY-IN-O3FD5RCBJHK4JCRC -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
-A ENN-PLY-IN-O3FD5RCBJHK4JCRC -m comment --comment "entry for podSelector" -j ENN-DPATCH-RCVFHUEC5K3IZT6D
-A ENN-PLY-IN-O3FD5RCBJHK4JCRC -m comment --comment "entry for namespaceSelector" -j ENN-DPATCH-DBBM6RRSD5ESYANP
-A ENN-PLY-IN-O3FD5RCBJHK4JCRC -m comment --comment "entry for ipBlock" -j ENN-DPATCH-ZDWBQO7N7BDCI4YX
-A ENN-PLY-IN-O3FD5RCBJHK4JCRC -m comment --comment "defualt reject rule" -j REJECT --reject-with icmp-port-unreachable
```

```
$ ipset list ENN-NS-ZC2OD5Z5N6SA7IXZ
  Name: ENN-NS-ZC2OD5Z5N6SA7IXZ
  Type: hash:ip
  Revision: 4
  Header: family inet hashsize 1024 maxelem 65536
  Size in memory: 272
  References: 2
  Members:
  10.244.1.128
  10.244.1.127
  10.244.2.124
 
```

```
$ ipset list ENN-NSSET-5STP5YUNBEIBJW6L
  Name: ENN-NSSET-5STP5YUNBEIBJW6L
  Type: hash:ip
  Revision: 4
  Header: family inet hashsize 1024 maxelem 65536
  Size in memory: 272
  References: 1
  Members:
  10.244.2.122

```

```
$ ipset list ENN-PODSET-T2GA36CFVTOK3KTQ
  Name: ENN-PODSET-T2GA36CFVTOK3KTQ
  Type: hash:ip
  Revision: 4
  Header: family inet hashsize 1024 maxelem 65536
  Size in memory: 176
  References: 1
  Members:
  10.244.2.124
  
```

#### namespace isolation daemon

> assume you have done step 1,2,3,4 in the above daemon

**_0. cleanup old networkPolicy_**

```
$ kubectl -n p-demo get networkPolicy
NAME           POD-SELECTOR   AGE
ingress-demo   <none>         3h
$ kubectl -n p-demo delete networkPolicy/ingress-demo
networkpolicy "ingress-demo" deleted
```

**_1. create new networkPolicy_**

- create new [ingress policy](./yaml/p-demo1-namespace-ingress.yaml), and new [egress policy](./yaml/p-demo1-namespace-egress.yaml)

> these two network policies isolate all pods from namespace "p-demo1"  
> which allow any pod in namespace "p-demo1" can visit itself  
> and allow connection from pods in any namespace with label "app=access"  
> and allow connection to pods in any namespace with label "app=access1"  

```yaml
$ cat p-demo1-namespace-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: namespace-ingress
  namespace: p-demo1
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: p-demo1
  - from:
    - namespaceSelector:
        matchLabels:
          app: access

```

```yaml
$ cat p-demo1-namespace-egress.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: namespace-egress
  namespace: p-demo1
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: p-demo1
  - to:
    - namespaceSelector:
        matchLabels:
          app: access1

```

- deploy 2 networkPolicies

```shell
$ kubectl create -f p-demo1-namespace-egress.yaml 
networkpolicy "namespace-egress" created
$ kubectl create -f p-demo1-namespace-ingress.yaml 
networkpolicy "namespace-ingress" created

```

**_2. check connection_**

- first check pod status

```
$ kubectl get pods -o wide --all-namespaces
p-demo         nginx-845466769c-k26sc                  1/1       Running   0          20h       10.244.1.127   ubuntu-client1
p-demo         nginx1-78bf97c65d-vw559                 1/1       Running   0          20h       10.244.2.124   ubuntu-client2
p-demo         nginx2-54c7bb866-gzksj                  1/1       Running   0          20h       10.244.1.128   ubuntu-client1
p-demo1        nginx-845466769c-h6wlk                  1/1       Running   0          20h       10.244.2.122   ubuntu-client2
p-demo2        nginx-845466769c-2mvhn                  1/1       Running   0          20h       10.244.1.129   ubuntu-client1
p-demo3        nginx-845466769c-88lmx                  1/1       Running   0          20h       10.244.2.123   ubuntu-client2
...
```

- check ingress connection

> pods in namespace "p-demo1" can be visited by pods in namespace "p-demo1"(itself) and pods in namespace "p-demo" and "p-demo2"(app-access)  
> but cannot be visited by other pods

```
$ kubectl -n p-demo1 exec nginx-845466769c-h6wlk -it /bin/bash
root@nginx-845466769c-h6wlk:/# echo > /dev/tcp/10.244.2.122/80
root@nginx-845466769c-h6wlk:/# exit

$ kubectl -n p-demo exec nginx-845466769c-k26sc -it /bin/bash
root@nginx-845466769c-k26sc:/# echo > /dev/tcp/10.244.2.122/80                                                                                                                                                   
root@nginx-845466769c-k26sc:/# exit

$ kubectl -n p-demo2 exec nginx-845466769c-2mvhn -it /bin/bash
root@nginx-845466769c-2mvhn:/# echo > /dev/tcp/10.244.2.122/80                                                                                                                                                   
root@nginx-845466769c-2mvhn:/# exit

$ kubectl -n p-demo3 exec nginx-845466769c-88lmx -it /bin/bash
root@nginx-845466769c-88lmx:/# echo > /dev/tcp/10.244.2.122/80                                                                                                                                                   
bash: connect: Connection refused
bash: /dev/tcp/10.244.2.122/80: Connection refused
root@nginx-845466769c-88lmx:/# exit
```

- check egress connection

> pods in namespace "p-demo1" can visit pods in namespace "p-demo1"(itself) and pods in namespace "p-demo3"(app-access1)  
> but cannot visit other pods

```
$ kubectl -n p-demo1 exec nginx-845466769c-h6wlk -it /bin/bash
root@nginx-845466769c-h6wlk:/# echo > /dev/tcp/10.244.2.122/80                                                                                                                                                   
root@nginx-845466769c-h6wlk:/# echo > /dev/tcp/10.244.2.123/80
root@nginx-845466769c-h6wlk:/# echo > /dev/tcp/10.244.2.124/80
bash: connect: Connection refused
bash: /dev/tcp/10.244.2.124/80: Connection refused
root@nginx-845466769c-h6wlk:/# echo > /dev/tcp/10.244.1.129/80
bash: connect: Connection refused
bash: /dev/tcp/10.244.1.129/80: Connection refused
root@nginx-845466769c-h6wlk:/# exit
```

