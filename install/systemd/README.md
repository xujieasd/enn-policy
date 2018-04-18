### This document will show how to create systemd service for enn-policy

0. OS: centos

1. create config file
```
$cat /etc/kubernetes/enn-policy
ENN_POLICY_ARGS="--kubeconfig /etc/kubernetes/local_kubeconfig --hostname-override 10.19.138.29 --log-dir=/var/log/kubernetes/enn-policy --ip-range=172.16.0.0/12"
ENN_POLICY_LOGS="--logtostderr=false --v=4"
```

2. create systemd start file
```
$cat /etc/systemd/system/enn-policy.service
[Unit]
Description=Kubernetes Network Policy Server
Documentation=https://10.19.248.12:30888/xujieasd/enn-policy
After=network.target kube-proxy.service
[Service]
EnvironmentFile=-/etc/kubernetes/enn-policy
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/opt/bin
ExecStart=/usr/bin/enn-policy \
     $ENN_POLICY_ARGS \
        $ENN_POLICY_LOGS
Restart=on-failure
LimitNOFILE=65536
StandardOutput=null
# StandardError=null
[Install]
WantedBy=multi-user.target
```

3. copy binary to directory /usr/bin
```
$ sudo cp enn-policy  /usr/bin/enn-policy
$ chmod 755 /usr/bin/enn-policy
```

4. create log directory
```
$ sudo mkdir /var/log/kubernetes/enn-policy
```

5. start your service
```
$ sudo systemctl enable enn-policy.service
$ sudo systemctl start enn-policy
```

6. check your service
````
$ systemctl status enn-policy
● enn-policy.service - Kubernetes Network Policy Server
  Loaded: loaded (/etc/systemd/system/enn-policy.service; enabled; vendor preset: disabled)
  Active: active (running) since 一 2018-03-26 14:34:54 HKT; 6s ago
    Docs: https://10.19.248.12:30888/xujieasd/enn-policy
Main PID: 32011 (enn-policy)
  CGroup: /system.slice/enn-policy.service
          └─32011 /usr/bin/enn-policy --kubeconfig /etc/kubernetes/local_kubeconfig --hostname-override 10.19.138.29 --log-dir=/var/log/kubernetes/enn-policy --ip-range=172.16.0.0/12 --logtostderr=false --v=4
```