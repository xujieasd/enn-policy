from kubernetes import client, config
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1DeleteOptions
from kubernetes.client import V1Namespace
from kubernetes.client import V1Pod
from kubernetes.client import V1Container
from kubernetes.client import V1ContainerPort
from kubernetes.client import V1PodSpec
from kubernetes.client import V1ResourceRequirements
from kubernetes.client import V1Probe
from kubernetes.client import V1ExecAction
from kubernetes.client import V1Service
from kubernetes.client import V1ServiceSpec
from kubernetes.client import V1ServicePort
from kubernetes.client import V1VolumeMount
from kubernetes.client import V1Volume
from kubernetes.client import V1HostPathVolumeSource
from kubernetes.client import V1RBDVolumeSource
from kubernetes.client import V1LocalObjectReference
from kubernetes.client import V1LimitRange
from kubernetes.client import V1LimitRangeSpec
from kubernetes.client import V1LimitRangeItem
from kubernetes.client import V1CephFSVolumeSource
from kubernetes.client import V1EmptyDirVolumeSource
from kubernetes.client import V1EnvVar
from kubernetes.client import V1beta1ReplicaSet
from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1LabelSelector
from kubernetes.client import V1beta1ReplicaSetSpec
from kubernetes.client import V1NetworkPolicy
from kubernetes.stream import stream
import urllib3
import re
from os.path import expanduser
import random


urllib3.disable_warnings()


class SimplePodInfo(object):
    def __init__(self, name, pod_ip, host_ip, status):
        self.name = name
        self.pod_ip = pod_ip
        self.host_ip = host_ip
        self.status = status


class K8SClient(object):
    user_configs = {
        "default": expanduser("~/.kube/config"),
    }

    @classmethod
    def register_user_config(cls, user, cluster_config):
        selected_config = cluster_config
        if type(cluster_config) is list:
            selected_config = random.sample(cluster_config, 1)[0]
        cls.user_configs[user] = expanduser(selected_config)

    def __init__(self, user="default"):
        # config.load_kube_config(expanduser("~/.kube/config"))
        # self.apiV1 = client.CoreV1Api()
        # self.apiV1beta1 = client.ExtensionsV1beta1Api()
        self.user = "Nobody"
        self.api_cache = {}
        self._load_user_config(user)

    def _load_user_config(self, user):
        if user not in self.user_configs:
            raise Exception("{} not found in configs".format(user))
        self.user = user
        if user not in self.api_cache:
            c = config.new_client_from_config(self.user_configs[self.user])
            #self.api_cache[user] = (client.CoreV1Api(c), client.ExtensionsV1beta1Api(c))
            self.api_cache[user] = (client.CoreV1Api(c), client.ExtensionsV1beta1Api(c), client.NetworkingV1Api(c))
        #self.apiV1, self.apiV1beta1 = self.api_cache[user]
        self.apiV1, self.apiV1beta1, self.networkingV1 = self.api_cache[user]

    def switch_user(self, user):
        if user == self.user:
            return
        self._load_user_config(user)

    def send_create_namespace_request(self, name, label={}):
        return self.apiV1.create_namespace(
            V1Namespace(
                metadata=V1ObjectMeta(
                    name=name,
                    labels=label,
                    deletion_grace_period_seconds=1,
                )
            )
        )

    def send_remove_namespace_request(self, name):
        return self.apiV1.delete_namespace(name, V1DeleteOptions())

    def get_all_namespaces(self):
        return [i.metadata.name for i in self.apiV1.list_namespace().items]

    def send_create_pod_request(
            self,
            namespace, name,
            image, args, ports={}, requests={}, limits={},
            probe="", probe_idelay=3, probe_period=3,
            node_selector=None,
            node_name=None,
            labels=None):
        metadata = V1ObjectMeta(name=name, namespace=namespace, labels=labels)
        ports = [V1ContainerPort(container_port=p, name=n) for p, n in ports.items()]
        probe_object = None
        if probe:
            probe_action = V1ExecAction(re.split(r" +", probe))
            probe_object = V1Probe(probe_action, initial_delay_seconds=probe_idelay, period_seconds=probe_period)
        container = V1Container(
            args=args.split(),
            image=image,
            name=name,
            ports=ports,
            resources=V1ResourceRequirements(requests=requests, limits=limits),
            liveness_probe=probe_object
        )
        spec = V1PodSpec(containers=[container],
                         node_selector=node_selector,
                         node_name=node_name,
                         restart_policy="Never")
        # {"kubernetes.io/hostname": "10.19.137.148"})
        pod = V1Pod(spec=spec, metadata=metadata)
        return self.apiV1.create_namespaced_pod(namespace, body=pod)

    def get_pod_info(self, namespace, name):
        return self.apiV1.read_namespaced_pod(name, namespace)

    def get_pods_info(self, namespace):
        return self.apiV1.list_namespaced_pod(namespace=namespace).items

    def send_remove_pod_request(self, namespace, name, throw_exp=True):
        try:
            r = self.apiV1.delete_namespaced_pod(name, namespace, V1DeleteOptions())
        except Exception as e:
            if throw_exp:
                raise e
            else:
                print e
        return r

    def clean_pods(self, namespace):
        return self.apiV1.delete_collection_namespaced_pod(namespace=namespace)

    def collect_pods_info(self, namespace):
        pod_list = self.apiV1.list_namespaced_pod(namespace=namespace)
        r = []
        for pod in pod_list.items:
            r.append(SimplePodInfo(pod.metadata.name, pod.status.pod_ip, pod.status.host_ip, pod.status.phase))
        return r

    def print_pod_stats(self, namespace):
        r = self.collect_pods_info(namespace)
        stats = {}
        for p in r:
            if p.host_ip not in stats:
                stats[p.host_ip] = {}
            if p.status not in stats[p.host_ip]:
                stats[p.host_ip][p.status] = 0
            stats[p.host_ip][p.status] += 1
        for h, s in stats.items():
            print h, s

    # node apis
    def list_ready_nodenames(self):
        r = []
        for n in self.apiV1.list_node().items:
            if n.spec.unschedulable:
                continue
            for c in n.status.conditions:
                if c.type == "Ready" and c.status == 'True':
                    r.append(n.metadata.name)
                    break
        return r

    def get_nodes(self):
        return self.apiV1.list_node().items

    # service apis
    def create_service(self, namespace, name, selector, port, protocol="TCP", service_type="ClusterIP"):
        metadata = V1ObjectMeta(name=name, namespace=namespace)
        port = V1ServicePort(name=name, port=port, protocol=protocol)
        spec = V1ServiceSpec(selector=selector, type=service_type, ports=[port])
        body = V1Service(metadata=metadata, spec=spec)
        return self.apiV1.create_namespaced_service(namespace, body)

    def remove_service(self, namespace, name, throw_exp=True):
        try:
            r = self.apiV1.delete_namespaced_service(name, namespace, V1DeleteOptions())
        except Exception as e:
            if throw_exp:
                raise e
            else:
                print e
        return r

    def get_service_info(self, namespace, name):
        return self.apiV1.read_namespaced_service(name, namespace)
    
    def list_services(self, namespace):
        return self.apiV1.list_namespaced_service(namespace)

    def clean_services(self, namespace):
        for s in self.list_services(namespace).items:
            self.remove_service(namespace, s.metadata.name)

    def pod_exec(self, namespace, pod_name, cmd, timeout=30):
        return self.apiV1.connect_get_namespaced_pod_exec(
            pod_name, namespace, command=cmd,
            stderr=True, stdin=False, stdout=True, tty=False,
            _request_timeout=timeout
        )
    
    def pod_exec_new(self, pod_name, namespace, command):
        resp = stream(self.apiV1.connect_get_namespaced_pod_exec, pod_name, namespace, command=command,
                      stderr=True, stdin=False, stdout=True, tty=False)
        return resp

    def apply_limit_range(self, namespace):
        name = "rlimit"
        limit_range = V1LimitRange(
            metadata=V1ObjectMeta(name=name, namespace=namespace),
            spec=V1LimitRangeSpec(limits=[V1LimitRangeItem(
                default={"cpu": "1", "memory": "4G"},
                default_request={"cpu": "300m", "memory": "500Mi"},
                type="Container"
            )])
        )
        if self.apiV1.list_namespaced_limit_range(namespace).items:
            return self.apiV1.replace_namespaced_limit_range(name=name, namespace=namespace, body=limit_range)
        else:
            return self.apiV1.create_namespaced_limit_range(namespace, body=limit_range)

    def tail_pod_logs(self, namespace, pod_name, line_count=None):
        if line_count:
            return self.apiV1.read_namespaced_pod_log(pod_name, namespace, tail_lines=line_count)
        return self.apiV1.read_namespaced_pod_log(pod_name, namespace)

    # endpoints
    def list_namespaced_endpoints(self, namespace):
        return self.apiV1.list_namespaced_endpoints(namespace).items

    def list_all_endpoints(self):
        return self.apiV1.list_endpoints_for_all_namespaces().items

    def list_all_services(self):
        return self.apiV1.list_service_for_all_namespaces().items

    # networkPolicy
    def list_namespaced_network_policy(self, namespace):
        return self.networkingV1.list_namespaced_network_policy(namespace).items
    
    def list_network_policy_for_all_namespaces(self):
        return self.networkingV1.list_network_policy_for_all_namespaces().items

    def remove_namespace_network_policy(self, namespace, name, throw_exp=True):
        try:
            r = self.networkingV1.delete_namespaced_network_policy(name, namespace, V1DeleteOptions())
        except Exception as e:
            if throw_exp:
                raise e
            else:
                print e
        return r
    
    def remove_namespace_network_policy_all(self, namespace, throw_exp=True):
        try:
            r = self.networkingV1.delete_collection_namespaced_network_policy(namespace)
        except Exception as e:
            if throw_exp:
                raise e
            else:
                print e
        return r
    
    def get_namespaced_network_policy(self, name, namespace):
        return self.networkingV1.read_namespaced_network_policy(name, namespace)

k8sclient = K8SClient()


class ComponentBuilder(object):
    def __init__(self, name, namespace, labels=None):
        self.meta = V1ObjectMeta(name=name, namespace=namespace, labels=labels)
        self.containers = []
        self.volumes = []
        self.target_labels = {}
        self.annotations = {}

    def add_container(self, name, image, args=None, requests={}, limits={}, probe="", volumes=[], ports=[], **envs):
        ports = [p.pod_port for p in ports]
        probe_object = None
        if probe:
            probe_action = V1ExecAction(re.split(r" +", probe))
            probe_object = V1Probe(probe_action, initial_delay_seconds=5, period_seconds=3)
        if args is not None:
            args = re.split(r" +", args)
        self.volumes.extend([v.volume for v in volumes if v.volume not in self.volumes])
        volume_mounts = [v.mount for v in volumes]
        container_env = [V1EnvVar(name=k, value=str(v)) for k, v in envs.items()]
        container = V1Container(
            args=args,
            image=image,
            name=name,
            ports=ports,
            resources=V1ResourceRequirements(requests=requests, limits=limits),
            liveness_probe=probe_object,
            volume_mounts=volume_mounts,
            env=container_env
        )
        self.containers.append(container)
        return self

    def attache_service(self, service):
        self.target_labels.update(service.selector)
        return self


class ServicePort(object):
    def __init__(self, name, container_port, port, protocol="TCP"):
        self.service_port = V1ServicePort(name=name, port=port, protocol=protocol)
        self.pod_port = V1ContainerPort(name=name, container_port=container_port)


class ServiceBuilder(object):
    def __init__(self, name, namespace, service_type="ClusterIP"):
        self.meta = V1ObjectMeta(name=name, namespace=namespace)
        self.ports = []
        self.service_type = service_type
        self.selector = {name + "-service": name}
        self.external_i_ps = []

    def add_port(self, port):
        print port
        if port not in self.ports:
            self.ports.append(port.service_port)
        return self
    
    def add_external_ip(self, ip):
        if ip not in self.external_i_ps:
            self.external_i_ps.append(ip)
        return self

    def deploy(self):
        spec = V1ServiceSpec(
            selector=self.selector,
            type=self.service_type,
            ports=self.ports,
            external_i_ps=self.external_i_ps
        )
        body = V1Service(metadata=self.meta, spec=spec)
        return k8sclient.apiV1.create_namespaced_service(self.meta.namespace, body)

    def un_deploy(self):
        return k8sclient.apiV1.delete_namespaced_service(
            self.meta.name,
            self.meta.namespace,
            V1DeleteOptions()
        )


class Volume(object):
    def __init__(self):
        self.mount = None
        self.volume = None


class HostPathVolume(Volume):
    def __init__(self, name, mount, path, read_only=True):
        self.mount = V1VolumeMount(name=name, mount_path=mount, read_only=read_only)
        self.volume = V1Volume(name=name, host_path=V1HostPathVolumeSource(path=path))


class EmptyDirVolume(Volume):
    def __init__(self, name, mount, read_only=False):
        self.mount = V1VolumeMount(name=name, mount_path=mount, read_only=read_only)
        self.volume = V1Volume(name=name, empty_dir=V1EmptyDirVolumeSource())


class RBDVolume(Volume):
    def __init__(self, name, mount, fs_type, image, monitors, pool, secret_name, sub_path, user="admin", read_only=False):
        self.mount = V1VolumeMount(name=name, mount_path=mount, read_only=read_only, sub_path=sub_path)
        self.volume = V1Volume(name=name, rbd=V1RBDVolumeSource(
            fs_type=fs_type,
            image=image,
            monitors=monitors.split(","),
            pool=pool,
            secret_ref=V1LocalObjectReference(secret_name),
            read_only=read_only,
            user=user
        ))


class CephFSVolume(Volume):
    def __init__(self, name, mount, monitors, secret_name, fs_path, sub_path, user="admin", read_only=False):
        self.mount = V1VolumeMount(name=name, mount_path=mount, read_only=read_only, sub_path=sub_path)
        self.volume = V1Volume(name=name, cephfs=V1CephFSVolumeSource(
            monitors=monitors.split(","),
            path=fs_path,
            secret_ref=V1LocalObjectReference(secret_name),
            read_only=read_only,
            user=user
        ))


class PodBuilder(ComponentBuilder):
    def __init__(self, *args):
        super(PodBuilder, self).__init__(*args)
        self.node_name = None

    def set_node(self, node_name):
        self.node_name = node_name
        return self

    def deploy(self, restart_policy="Never"):
        spec = V1PodSpec(
            containers=self.containers,
            node_name=self.node_name,
            volumes=self.volumes,
            restart_policy=restart_policy
        )
        self.meta.labels = self.target_labels
        pod = V1Pod(spec=spec, metadata=self.meta)
        return k8sclient.apiV1.create_namespaced_pod(self.meta.namespace, body=pod)

    def un_deploy(self):
        return k8sclient.apiV1.delete_namespaced_pod(
            self.meta.name,
            self.meta.namespace,
            V1DeleteOptions()
        )


class ReplicaSetBuilder(ComponentBuilder):
    def __init__(self, *args):
        super(ReplicaSetBuilder, self).__init__(*args)
        self.replicas = 1
        rs_marks = {"replicaset": self.meta.name}
        self.selector = V1LabelSelector(
            match_labels=rs_marks
        )
        self.target_labels.update(rs_marks)

    def set_hostname(self, hostname):
        self.annotations['pod.beta.kubernetes.io/hostname'] = hostname
        self.replicas = 1
        return self

    def replicas(self, count):
        self.replicas = count
        #
        if 'pod.beta.kubernetes.io/hostname' in self.annotations:
            del self.annotations['pod.beta.kubernetes.io/hostname']
        return self

    def _build_rs(self):
        pod_spec = V1PodSpec(
            containers=self.containers,
            volumes=self.volumes
        )
        template = V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                labels=self.target_labels,
                annotations=self.annotations or None
            ),
            spec=pod_spec
        )
        rs_spec = V1beta1ReplicaSetSpec(
            replicas=self.replicas,
            selector=self.selector,
            template=template
        )
        rs = V1beta1ReplicaSet(
            metadata=self.meta,
            spec=rs_spec
        )
        return rs

    def deploy(self):
        return k8sclient.apiV1beta1.create_namespaced_replica_set(
            self.meta.namespace,
            body=self._build_rs()
        )

    def un_deploy(self):
        pods = k8sclient.collect_pods_info(self.meta.namespace)
        for pod in pods:
            if pod.name.find(self.meta.name) != -1:
                k8sclient.send_remove_pod_request(self.meta.namespace, pod.name)

        return k8sclient.apiV1beta1.delete_namespaced_replica_set(
            name=self.meta.name,
            namespace=self.meta.namespace,
            body=V1DeleteOptions()
        )

