# Kind: VNet
VNet is the CRD implementation of a Netris-Operator.

Register the VNet [kind](https://github.com/netrisai/netris-operator/tree/dev/deploy) in the Kubernetes cluster before creating VNet objects.

### VNet Attributes

```
apiVersion: k8s.netris.ai/v1alpha1
kind: VNet
metadata:
  name: myVnet
spec:
  owner: admin                                       # [1]
  state: active                                      # [2]
  guestTenants: []                                   # [3]
  sites:                                             # [4]
  - name: yerevan                                    # [5]
    gateways:                                        # [6]
    - gateway4: 109.23.0.6/24
    - gateway4: 109.24.0.6/24
    - gateway6: 2001:db8:acad::fffe/64
    switchPorts:                                     # [7]
    - name: swp4@rlab-leaf1                          # [8]
      vlanId: 1050                                   # [9]
    - name: swp7@rlab-leaf1
      portIsUntagged: true                           # [10]
      state: disable                                 # [11]
```

Ref | Attribute                              | Default     | Description
----| -------------------------------------- | ----------- | ----------------
[1] | owner                                  | ""          | Users with permission to owner tenant can manage parameters of the V-Net as well as add/edit/remove ports assigned to any of tenants where user has permission.
[2] | state                                  | ""          | V-Net state. Allowed values: `active` or `disable`. 
[3] | guestTenants                           | []          | List of tenants allowed to add/edit/remove ports to the V-Net but not allowed to manage other parameters of the circuit.
[4] | sites                                  | []          | List of sites. Ports from these sites will be allowed to participate to the V-Net. Multi-site circuits are possible for sites connected through a backbone port.
[5] | sites[n].name                          | ""          | Site's name.
[6] | sites[n].gateways                      | []          | List of gateways. Possible keys in the list: `gateway4` or `gateway6`. Selected address will be serving as anycast default gateway for selected subnet. In case of multi-site V-Net, multi-site subnet should be configured under Subnets section.
[7] | sites[n].switchPorts                   | []          | List of switchPorts.
[8] | sites[n].switchPorts[n].name           | ""          | SwitchPorts name.
[9] | sites[n].switchPorts[n].vlanId         | nil         | VLAN tag for current port.
[10]| sites[n].switchPorts[n].portIsUntagged | false       | Untag for sending frames towards this port without VLAN tag. Ignored if `vlanId` is set. Allowed values: `true` or `false`. 
[11]| sites[n].switchPorts[n].state          | active      | Port state. Allowed values: `active` or `disable`. 
