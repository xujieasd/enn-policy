"""
This script is used for testing network bandwidth left for sending traffic in a single channel.
"""
from k8sclient.keywords import (
    list_ready_nodes,
    get_pod_ip,
    wait_for_pod_state,
    RUNNING,
    SUCCEEDED,
    tail_pod_logs,
    delete_pod,
    remove_pod,
    NOT_FOUND,
    register_cluster,
    switch_cluster,
    cleanup_pods,
    cleanup_services,
)
from k8sclient.Components import PodBuilder, ServicePort, ServiceBuilder
import re
import time
import sys
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("--nodePre", type=str, help="The prefix of nodes, e.g run 'kubectl get node', if you get node name is 'ubuntu-clientX' then value here is 'ubuntu-client'.")
args = parser.parse_args()

image = "127.0.0.1:30100/library/alpine-iperf:0.1"
server_args = "iperf -f M -i 1 -m -s"
client_args = r"iperf -f M -t 10 -i 1 -c %s"
namespace = "k8sft"
nodes = sorted(list_ready_nodes())
nodes = [n for n in nodes if n.startswith(args.nodePre)]
server_port = ServicePort("serverport", 5001, 5001)
global_server_name = "iperf-server"
glimit = {'cpu': '0', 'memory': '8Gi'}
grequest = {'cpu': '0', 'memory': '0'}
server_service = ServiceBuilder(global_server_name, namespace).add_port(server_port)
reports = {}
report_css = """<style>
table, th, td {
    border: 1px solid black;
    border-collapse: collapse;
}
tr:nth-child(even) {background: #CCC}
tr:nth-child(odd) {background: #FFF}
</style>
"""
report_title = r"""<H1>Pod to Pod network throughput, single connection. (MBytes/sec)</H1>
Cell format: <b><i>[bw via ip]|[bw via service]</i></b>
<br>
CPU limit: <b>no limit</b>
<br>
Memory Limit: <b>8Gi</b>
<br>
Server cmd: <b>%s</b>
<br>
Client cmd: <b>%s</b>
<br>
<br>
""" % (server_args, client_args)


def add_report(src, dest, result):
    if src not in reports:
        reports[src] = {}
    if dest not in reports[src]:
        reports[src][dest] = result
    else:
        reports[src][dest] = "|".join([reports[src][dest], result])


def save_report():
    shorts = {}
    for n in nodes:
        shorts[n] = n.split(".")[-1]
    with open("report.html", "w") as f:
        f.write(report_css)
        f.write(report_title)
        f.write("<table><tr><th>From/To</th>\n")
        for n in nodes:
            f.write("<th>{}</th>\n".format(shorts[n]))
        f.write("</tr>\n")
        for n in nodes:
            if n not in reports:
                continue
            f.write("<tr><td>{}</td>\n".format(shorts[n]))
            for m in nodes:
                if m not in reports[n]:
                    f.write("<td>-</td>\n")
                else:
                    f.write("<td>{}</td>\n".format(reports[n][m]))
            f.write("</tr>\n")
        f.write("</table>\n")


def test(server_node, client_node):
    print client_node, "->", server_node
    server_pod_name = "server-" + "-".join(server_node.split("."))
    PodBuilder(
        server_pod_name, namespace
    ).add_container(
        name=server_pod_name + "-container",
        image=image,
        args=server_args,
        limits=glimit,
        requests=grequest,
        ports=[server_port]
    ).set_node(server_node).attache_service(server_service).deploy()
    # wait for server pod running
    wait_for_pod_state(namespace, server_pod_name, timeout=600, expect_status=RUNNING)
    time.sleep(3)
    # get server pod ip
    server_pod_ip = get_pod_ip(namespace, server_pod_name)
    run_client(client_node, server_node, server_pod_ip)
    run_client(client_node, server_node, global_server_name)
    delete_pod(namespace, server_pod_name)
    wait_for_pod_state(namespace, server_pod_name, timeout=240, expect_status=NOT_FOUND)


def run_server(server_node):
    server_pod_name = "server-" + "-".join(server_node.split("."))
    PodBuilder(
        server_pod_name, namespace
    ).add_container(
        name=server_pod_name + "-container",
        image=image,
        args=server_args,
        limits=glimit,
        requests=grequest,
        ports=[server_port]
    ).set_node(server_node).attache_service(server_service).deploy()
    try:
        # wait for server pod running
        wait_for_pod_state(namespace, server_pod_name, timeout=600, expect_status=RUNNING)
        time.sleep(5)
        # get server pod ip
        server_pod_ip = get_pod_ip(namespace, server_pod_name)
        for node in nodes:
            run_client(node, server_node, server_pod_ip)
            run_client(node, server_node, global_server_name)
    except:
        pass
    finally:
        remove_pod(namespace, server_pod_name)


def run_client(client_node, server_node, server_pod_ip):
    client_pod_name = "client-" + "-".join(client_node.split("."))
    pod = PodBuilder(
        client_pod_name, namespace
    ).add_container(
        name=client_pod_name + "-container",
        image=image,
        args=client_args % server_pod_ip,
        limits=glimit,
        requests=grequest,
    ).set_node(client_node)
    try:
        for i in range(2):
            pod.deploy()
            # wait for client complete
            wait_for_pod_state(namespace, client_pod_name, timeout=600, expect_status=SUCCEEDED)
            # parse client log to get tx speed.
            logs = tail_pod_logs(namespace, client_pod_name, lines=20).strip()
            print logs
            summary = logs.split("\n")[-1]
            m = re.match(r".*[^.\d]+([.\d]+) MBytes/sec", summary)
            if m:
                break
            remove_pod(namespace, client_pod_name)
        if m:
            print server_node, client_node, server_pod_ip, m.group(1)
            add_report(client_node, server_node, m.group(1))
        else:
            add_report(client_node, server_node, summary)
    except:
        pass
    finally:
        remove_pod(namespace, client_pod_name)


def cleanup():
    cleanup_pods(namespace=namespace)
    cleanup_services(namespace)


cleanup()
server_service.deploy(force=True)
# test all node pair
for node in nodes:
    run_server(node)
# test random pair of node

save_report()
server_service.un_deploy()
