from k8sclient.Components import (
    ServicePort,
    PodBuilder,
    ServiceBuilder,
)

from k8sclient.keywords import (
    list_ready_nodes,
    is_pod_running,
    get_pod_ip,
    create_namespace,
)

import argparse
import time


parser = argparse.ArgumentParser()
parser.add_argument("--namespaceNumber", type=str, help="How many namespace will created.")
parser.add_argument("--nodePre", type=str, help="The prefix of nodes, e.g run 'kubectl get node', if you get node name is 'ubuntu-clientX' then value here is 'ubuntu-client'.")
args = parser.parse_args()

nodes = list_ready_nodes()
nodes = [n for n in nodes if n.startswith(args.nodePre)]
image = "xujieasd/alphine-restcheck:0.5"
client_port = ServicePort("clientport", 8080, 8080)
counts = [0] * len(nodes)
readys = [True] * len(nodes)
dones = [False] * len(nodes)
node_marks = ["-".join(n.split(".")) for n in nodes]

NS_NUMBER = int(args.namespaceNumber)

def create_numbered_namespace():
    for i in range (0, NS_NUMBER):
        namespace_name = ("namespace-%d" % i)
        label = ("ns-%d" % i)
        namespace_label = {label:label}
        create_namespace(namespace_name, namespace_label)

def create_pods_services_peer_node():
    total = 0
    start = time.time()
    while not all(dones):
        # deploy ready nodes
        for i, n in enumerate(nodes):
            namespace_id = counts[i]
            namespace_name = ("namespace-%d" % namespace_id)
            if dones[i]:
                continue
            pod_name = ("pod-%s-%d" % (node_marks[i], namespace_id))
            if readys[i]:
                # create a new pod
                inter_pod = PodBuilder(
                    pod_name,
                    namespace_name,
                ).set_node(
                    n
                ).add_container(
                    pod_name,
                    image=image,
                    # args=args,
                    ports=[client_port],
                    requests={'cpu': '0', "memory": '0'},
                    limits = {'cpu': '1', "memory": '32Mi'}
                )
                svc_name = ("svc-%s-%d" % (node_marks[i], namespace_id))
                inter_svc = ServiceBuilder(svc_name, namespace_name).add_port(client_port)
                inter_svc.deploy(force=True)
                inter_pod.attache_service(inter_svc)
                inter_pod.deploy()
                readys[i] = False
                # print "creating", pod_name
            else:
                # check for current pod running
                readys[i] = is_pod_running(namespace_name, pod_name)
                if readys[i]:
                    total += 1
                    counts[i] += 1
                if counts[i] >= NS_NUMBER:
                    print "It took %ds to deploy %d pod on %s" % (int(time.time()-start), NS_NUMBER, n)
                    # print n, "is done~!", "total", total
                    dones[i] = True
        time.sleep(3)
    print "it took", time.time() - start, "s"

def create():
    create_numbered_namespace()
    create_pods_services_peer_node()

create()