### Example Daemon

- this document will show some examples to see how enn-policy works
- before go through blew contents, you must have [deployed enn-policy](./user-guide.md) in your k8s environment
- also, you need to have a brief understanding of [kubernetes network policy](https://kubernetes.io/docs/concepts/services-networking/network-policies/)

#### Create simple daemon

**_1. config new namespace_**

we will first create a new namespace, then we will operate the daemon within this namespace

```
$ kubectl create ns p-demo
```

**_2. create some pods_**

we will create some nginx pods in the p-demo namespace with different labels  
use deployment to deploy these pods

- create pod with label app=nginx, you can find daemon yaml file [here](./yaml/nginx-policy.yaml)
 
```
$ kubectl create -f nginx-policy.yaml
```

- create pod with label app=access, you can find daemon yaml file [here](./yaml/nginx-policy-access.yaml)

```
$ kubectl create -f nginx-policy-access.yaml
```

- create pod with label app=reject, you can find daemon yaml file [here](./yaml/nginx-policy-reject.yaml)

```
$ kubectl create -f nginx-policy-reject.yaml
```

- check you've created these pods in namespace p-demo

```
$ kubectl get pods -o wide -n p-demo
NAME                            READY     STATUS    RESTARTS   AGE       IP             NODE
nginx-8556675ff6-ccjn5          1/1       Running   0          10m       10.244.2.119   ubuntu-client2
nginx-access-5b776b9cdd-nnbth   1/1       Running   0          2s        10.244.2.120   ubuntu-client2
nginx-reject-789bb6f89b-4x6l5   1/1       Running   0          3m        10.244.1.125   ubuntu-client1

```

**_3. check network traffic before enable policy_**

By default, every pods should be accessed by every pods in k8s environment  
so pod _nginx-8556675ff6-ccjn5_ can be accessed by both pod _nginx-access-5b776b9cdd-nnbth_ and pod _nginx-reject-789bb6f89b-4x6l5_  
We use **echo > /dev/tcp/ip/port** to check whether tcp traffic can be accessible, if traffic is accessible, it will output nothing

- check pod _nginx-access-5b776b9cdd-nnbth_ can access pod _nginx-8556675ff6-ccjn5_

```
$ kubectl -n p-demo exec nginx-access-5b776b9cdd-nnbth -it /bin/bash
$ echo > /dev/tcp/10.244.2.119/80
$ exit
```

- check pod _nginx-reject-789bb6f89b-4x6l5_ can access pod _nginx-8556675ff6-ccjn5_

```
$ kubectl -n p-demo exec nginx-reject-789bb6f89b-4x6l5 -it /bin/bash
$ echo > /dev/tcp/10.244.2.119/80
$ exit
```

**_4. deploy a simple network policy_**

create a network policy like below, you can also find daemon yaml file [here](./yaml/policy-pod-acess-ingress.yaml)  
which allow traffic only from pods with labels app=acess in namespace p-demo to pods with label app=nginx in namespace p-demo

```yaml
$ cat policy-pod-access-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: pod-access-ingress
  namespace: p-demo
spec:
  podSelector:
    matchLabels:
      app: nginx
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: access

```

deploy this networkPolicy

```
$ kubectl create -f policy-pod-access-ingress.yaml
```

**_5. check network traffic after enable policy_**

with policy deployed in step 4,  
now pod _nginx-access-5b776b9cdd-nnbth_ can access pod _nginx-8556675ff6-ccjn5_  
and pod _nginx-reject-789bb6f89b-4x6l5_ cannot access pod _nginx-8556675ff6-ccjn5_

- check pod _nginx-access-5b776b9cdd-nnbth_

```
$ kubectl -n p-demo exec nginx-access-5b776b9cdd-nnbth -it /bin/bash
$ echo > /dev/tcp/10.244.2.119/80
$ exit
```
- check pod _nginx-reject-789bb6f89b-4x6l5_

```
$ kubectl -n p-demo exec nginx-reject-789bb6f89b-4x6l5 -it /bin/bash
$ echo > /dev/tcp/10.244.2.119/80
bash: connect: Connection refused
bash: /dev/tcp/10.244.2.119/80: Connection refused
$ exit
```

**so network policy works!**  
[see advanced demo](./advaned-daemon.md)