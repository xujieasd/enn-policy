from k8sclient.keywords import (
    list_ready_nodes,
    pod_exec_new,
    get_service,
    get_pod_ip,
)

import time
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("--namespaceNumber", type=str, help="How many namespace will be created.")
parser.add_argument("--policyNumber", type=str, help="How many policies will be created for every policy.")
parser.add_argument("--nodePre", type=str, help="The prefix of nodes, e.g run 'kubectl get node', if you get node name is 'ubuntu-clientX' then value here is 'ubuntu-client'.")
args = parser.parse_args()

nodes = list_ready_nodes()
node_marks = ["-".join(n.split(".")) for n in nodes]
nodes = [n for n in nodes if n.startswith(args.nodePre)]
NS_NUMBER = int(args.namespaceNumber)
PL_NUMBER = int(args.policyNumber)

SUCCESS_MARK = "CHECK_PASS"

def _get_pod_ip(namespace, pod):
    return get_pod_ip(namespace, pod)

def _check_service_connected(namespace, pod, service, port):
    s = time.time()
    o = pod_exec_new(namespace, pod, ["/opt/check2.sh", service, port])
    if o.find(SUCCESS_MARK) == -1:
        print "Fail to check %s:%s on %s, error message: [%s]. it took %s" % (service, port, pod, o, str(time.time() - s))
        return False
    return True

def _check_service_unconnected(namespace, pod, service, port):
    o = pod_exec_new(namespace, pod, ["/opt/check2.sh", service, port])
    if o.find(SUCCESS_MARK) != -1:
        print "Fail to check %s:%s on %s, error message: [%s]. expected unconnected " % (service, port, pod, o)
        return False
    return True

def check_service_connected(namespace, pod, service, port):
    # give it a retry
    return _check_service_connected(namespace, pod, service, port) or \
           _check_service_connected(namespace, pod, service, port)

def check_service_unconnected(namespace, pod, service, port):
    return _check_service_unconnected(namespace, pod, service, port)

def check_namespaced_pod_connection(namespace_id, namespace_set):

    err = []

    for nj, namespace_j in enumerate(namespace_set):
        namespace_name = ("namespace-%d" % namespace_j)
        for i, ni in enumerate(nodes):

            pod_name = ("pod-%s-%d" % (node_marks[i], namespace_j))
            #print "namespace:%s pod_name: %s" % (namespace_name,pod_name)

            for k, nk in enumerate(nodes):
                if nk == ni and namespace_id == namespace_j:
                    continue
                svc_name = ("svc-%s-%d" % (node_marks[k], namespace_id))
                svc_namespace = ("namespace-%d" % namespace_id)
                svc = get_service(svc_namespace,svc_name)
                svc_ip = svc.spec.cluster_ip
                #print "svc_name: %s, ip: %s, namespace: %s" % (svc_name, svc_ip, namespace_id)

                if not check_service_connected(namespace_name, pod_name, svc_ip, "8080"):
                    err_log = "Fail to connect %s:%s on %s." % (svc_name,"8080",pod_name)
                    err.append(err_log)

    return err

def check_namespace_pod_unconnection(namespace_id, namespace_set):

    err = []

    for nj, namespace_j in enumerate(namespace_set):
        namespace_name = ("namespace-%d" % namespace_j)
        for i, ni in enumerate(nodes):

            pod_name = ("pod-%s-%d" % (node_marks[i], namespace_j))
            #print "namespace:%s pod_name: %s" % (namespace_name,pod_name)

            for k, nk in enumerate(nodes):
                if namespace_id == namespace_j:
                    continue
                svc_name = ("svc-%s-%d" % (node_marks[k], namespace_id))
                svc_namespace = ("namespace-%d" % namespace_id)
                svc = get_service(svc_namespace,svc_name)
                svc_ip = svc.spec.cluster_ip
                #print "svc_name: %s, ip: %s, namespace: %s" % (svc_name, svc_ip, namespace_id)

                if not check_service_unconnected(namespace_name, pod_name, svc_ip, "8080"):
                    err_log = "unexpected connection %s:%s on %s." % (svc_name,"8080",pod_name)
                    err.append(err_log)
    return err

def check_all_pod_connection():
    err = []
    print "check connection"
    for i in range (0, NS_NUMBER):
        print "check namespace %d" % i
        namespace_set = []
        for j in range (0, PL_NUMBER):
            namespace_set.append(j)
        r = check_namespaced_pod_connection(i, namespace_set)
        if r:
            for i, rr in enumerate(r):
                err.append(rr)
    if err:
        print "connection issue found."
        for i, r in enumerate(err):
            print r
    else:
        print "no issue found"
    return err

def check_all_pod_unconnection():
    err = []
    print "check unconnection"
    for i in range (0, NS_NUMBER):
        print "check namespace %d" % i
        namespace_set = []
        for j in range (PL_NUMBER, NS_NUMBER):
            namespace_set.append(j)
        r = check_namespace_pod_unconnection(i, namespace_set)
        if r:
            for i, rr in enumerate(r):
                err.append(rr)
    if err:
        print "unconnection issue found."
        for i, r in enumerate(err):
            print r
    else:
        print "no issue found"
    return err

def check_all():
    err = []
    r1 = check_all_pod_connection()
    if r1:
        for i, rr in enumerate(r1):
            err.append(rr)
    r2 = check_all_pod_unconnection()
    if r2:
        for i, rr in enumerate(r2):
            err.append(rr)
    if err:
        print "issue found."
        for i, r in enumerate(err):
            print r
    else:
        print "no issue found"
    return err

check_all()
