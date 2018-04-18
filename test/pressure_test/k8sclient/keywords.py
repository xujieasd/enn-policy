from .K8SClient import k8sclient
from kubernetes.client.rest import ApiException
import time

def wait_for_pod_state(namespace, name, timeout, expect_status):
    time.sleep(1)
    while timeout > 0:
        try:
            r = k8sclient.get_pod_info(namespace, name)
            if r.status.phase == expect_status:
                break
        except ApiException as e:
            if expect_status == NOT_FOUND and e.status == 404:
                break
            else:
                raise e
        time.sleep(1)
        timeout -= 1
    assert timeout > 0, "Fail to wait expect status %s, current %s" % (expect_status, r.status.phase)
    return timeout


def is_pod_running(namespace, pod):
    try:
        return k8sclient.get_pod_info(namespace, pod).status.phase == RUNNING
    except ApiException:
        return False


def delete_pod(namespace, name):
    return k8sclient.send_remove_pod_request(namespace, name)


def remove_pod(namespace, name, timeout=120):
    try:
        k8sclient.send_remove_pod_request(namespace, name)
    except ApiException as e:
        if e.status == 404:
            return
        raise e
    wait_for_pod_state(namespace, name, timeout=timeout, expect_status=NOT_FOUND)


def delete_service(namespace, name):
    return k8sclient.remove_service(namespace, name)


def cleanup_pods(namespace):
    return k8sclient.clean_pods(namespace=namespace)


def cleanup_services(namespace):
    return k8sclient.clean_services(namespace)


def list_ready_nodes():
    return k8sclient.list_ready_nodenames()


def list_nodes():
    return k8sclient.get_nodes()


def get_pod_ip(namespace, pod_name):
    return k8sclient.get_pod_info(namespace, pod_name).status.pod_ip

def get_service_ip(namespace, service_name):
    return k8sclient.get_service_info(namespace, service_name).spec.cluster_ip

def get_service(namespace, service_name):
    return k8sclient.get_service_info(namespace, service_name)

def pod_exec(namespace, pod_name, cmd, timeout=30, client=None):
    if client is None:
        client = k8sclient
    r = client.pod_exec(namespace, pod_name, cmd, timeout=timeout)
    return r

def pod_exec_new(namespace, pod_name, cmd):
    r = k8sclient.pod_exec_new(pod_name, namespace, cmd)
    return r

def tail_pod_logs(namespace, pod_name, lines=None):
    return k8sclient.tail_pod_logs(namespace, pod_name, lines)


def list_namespaced_pods(namespace):
    return k8sclient.get_pods_info(namespace)


def list_namespaces():
    return k8sclient.get_all_namespaces()

def create_namespace(name, label={}):
    return k8sclient.send_create_namespace_request(name, label)

def remove_namespace(name):
    return k8sclient.send_remove_namespace_request(name)

def list_namespaced_services(namespace):
    return k8sclient.list_services(namespace).items


def list_all_services():
    return k8sclient.list_all_services()


def list_namespaced_endpoints(namespace):
    return k8sclient.list_namespaced_endpoints(namespace)


def list_all_endpoints():
    return k8sclient.list_all_endpoints()


# cluster management
def register_cluster(cluster_name, config_file):
    k8sclient.register_user_config(cluster_name, config_file)


def switch_cluster(cluster_name):
    k8sclient.switch_user(cluster_name)

# networkPolicy
def list_namespaced_network_policy(namespace):
    return k8sclient.list_namespaced_network_policy(namespace)

def list_network_policy_for_all_namespaces():
    return k8sclient.list_network_policy_for_all_namespaces()

def delete_namespaced_network_policy_all(namespace):
    return k8sclient.remove_namespace_network_policy_all(namespace)

def read_namespaced_network_policy(name, namespace):
    return k8sclient.get_namespaced_network_policy(name, namespace)

RUNNING = "Running"
NOT_FOUND = "404"
SUCCEEDED = "Succeeded"
