#!/usr/bin/env python3

import argparse
import yaml

fullname = '{{ include "netris-operator.fullname" . }}'
namespace = '{{ include "netris-operator.namespace" . }}'
service_account_name = '{{ include "netris-operator.serviceAccountName" . }}'


def main():
    args = args_parser()
    with open(args.y, 'r') as f:
        yaml_items = yaml.safe_load(f)
    kind = ''
    try:
        kind = yaml_items['kind']
    except KeyError:  # not k8s resource
        exit(1)
    if 'creationTimestamp' in yaml_items['metadata']:
        del yaml_items['metadata']['creationTimestamp']
    if kind == 'Role':
        role(yaml_items)
    elif kind == 'ClusterRole':
        cluster_role(yaml_items)
    elif kind == 'RoleBinding':
        role_binding(yaml_items)
    elif kind == 'ClusterRoleBinding':
        cluster_role_binding(yaml_items)
    else:  # unexpected resource kind
        exit(1)


def args_parser():
    parser = argparse.ArgumentParser(description='rbac to helm template script')
    parser.add_argument('y', metavar='$1', type=str, help='yaml file path')
    args = parser.parse_args()
    return args


def role(_yaml):
    name = _yaml['metadata']['name']
    _yaml['metadata']['name'] = '{}-{}'.format(fullname, name)
    _yaml['metadata']['namespace'] = namespace
    print(_yaml)


def role_binding(_yaml):
    name = _yaml['metadata']['name']
    role_ref_name = _yaml['roleRef']['name']
    _yaml['metadata']['name'] = '{}-{}'.format(fullname, name)
    _yaml['metadata']['namespace'] = namespace
    _yaml['roleRef']['name'] = '{}-{}'.format(fullname, role_ref_name)
    for key, value in enumerate(_yaml['subjects']):
        if value['kind'] == 'ServiceAccount':
            _yaml['subjects'][key]['name'] = service_account_name
            _yaml['subjects'][key]['namespace'] = namespace
    print(_yaml)


def cluster_role(_yaml):
    name = _yaml['metadata']['name']
    _yaml['metadata']['name'] = '{}-{}'.format(fullname, name)
    print(_yaml)


def cluster_role_binding(_yaml):
    name = _yaml['metadata']['name']
    role_ref_name = _yaml['roleRef']['name']
    _yaml['metadata']['name'] = '{}-{}'.format(fullname, name)
    _yaml['roleRef']['name'] = '{}-{}'.format(fullname, role_ref_name)
    for key, value in enumerate(_yaml['subjects']):
        if value['kind'] == 'ServiceAccount':
            _yaml['subjects'][key]['name'] = service_account_name
            _yaml['subjects'][key]['namespace'] = namespace
    print(_yaml)


if __name__ == '__main__':
    main()
