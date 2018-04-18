from k8sclient.Components import (
    NetworkPolicyEgressRule,
    NetworkPolicyIngressRule,
    NetworkPolicyBuilder,
)

import argparse

parser = argparse.ArgumentParser()
parser.add_argument("--namespaceNumber", type=str, help="How many namespace will be created.")
parser.add_argument("--policyNumber", type=str, help="How many policies will be created for every policy.")
args = parser.parse_args()

NS_NUMBER = int(args.namespaceNumber)
PL_NUMBER = int(args.policyNumber)

def create_numbered_network_policy():
    for i in range (0, NS_NUMBER):
        namespace_name = ("namespace-%d" % i)
        policy_name = ("ingress-policy-l%d" % i)
        label = ("ns-%d" % i)
        namespace_label_local = {label:label}
        ingress_rule_l = NetworkPolicyIngressRule()
        ingress_rule_l.add_policy_namespace_selector(namespace_label=namespace_label_local)

        ingress_policy_l = NetworkPolicyBuilder(
            policy_name,
            namespace_name,
        ).add_ingress_rule(
            ingress_rule_l,
        ).add_policy_type(
            "Ingress",
        )
        ingress_policy_l.deploy()

        policy_name = ("ingress-policy-%d" % i)
        ingress_rule = NetworkPolicyIngressRule()

        for j in range (0, PL_NUMBER):
            label = ("ns-%d" % j)
            namespace_label_local = {label:label}
            ingress_rule.add_policy_namespace_selector(namespace_label=namespace_label_local)

        ingress_policy = NetworkPolicyBuilder(
            policy_name,
            namespace_name,
        ).add_ingress_rule(
            ingress_rule,
        ).add_policy_type(
            "Ingress",
        )
        ingress_policy.deploy()

create_numbered_network_policy()