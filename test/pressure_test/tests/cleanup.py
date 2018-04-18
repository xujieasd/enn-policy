from k8sclient.keywords import (
    cleanup_pods,
    cleanup_services,
    remove_namespace,
    delete_namespaced_network_policy_all,
    list_namespaces,
)

def get_namespace():
    result = 0
    namespaces = list_namespaces()
    for i, name in enumerate(namespaces):
        if name.startswith("namespace-"):
            result = result + 1
    return result

def delete_numbered_namespace(ns_number):
    for i in range (0, ns_number):
        namespace_name = ("namespace-%d" % i)
        remove_namespace(namespace_name)

def cleanup_all_pods(ns_number):
    for i in range (0, ns_number):
        namespace_name = ("namespace-%d" % i)
        cleanup_pods(namespace_name)

def cleanup_all_services(ns_number):
    for i in range (0, ns_number):
        namespace_name = ("namespace-%d" % i)
        cleanup_services(namespace_name)

def cleanup_all_network_policy(ns_number):
    for i in range (0, ns_number):
        namespace_name = ("namespace-%d" % i)
        delete_namespaced_network_policy_all(namespace_name)

def cleanup():
    ns_number = get_namespace()
    cleanup_all_network_policy(ns_number)
    cleanup_all_services(ns_number)
    cleanup_all_pods(ns_number)
    delete_numbered_namespace(ns_number)

cleanup()
