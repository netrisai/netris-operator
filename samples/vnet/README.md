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
  ownerTenant: admin                                 # [1]
  guestTenants: []                                   # [2]
  sites:                                             # [3]
  - name: yerevan                                    # [4]
    gateways:                                        # [5]
    - gateway4: 109.23.0.6/24
    - gateway4: 109.24.0.6/24
    - gateway6: 2001:db8:acad::fffe/64
    switchPorts:                                     # [6]
    - name: swp4@rlab-leaf1                          # [7]
      vlanId: 1050                                   # [8]
    - name: swp7@rlab-leaf1
      state: disable                                 # [9]
```

Ref | Attribute                              | Default     | Description
----| -------------------------------------- | ----------- | ----------------
[1] | ownerTenant                            | ""          | Users with permission to owner tenant can manage parameters of the V-Net as well as add/edit/remove ports assigned to any of tenants where user has permission.
[2] | guestTenants                           | []          | List of tenants allowed to add/edit/remove ports to the V-Net but not allowed to manage other parameters of the circuit.
[]  | state                                  | active      | V-Net state. Allowed values: `active` or `disable`. 
[3] | sites                                  | []          | List of sites. Ports from these sites will be allowed to participate to the V-Net. Multi-site circuits are possible for sites connected through a backbone port.
[4] | sites[n].name                          | ""          | Site's name.
[5] | sites[n].gateways                      | []          | List of gateways. Possible keys in the list: `gateway4` or `gateway6`. Selected address will be serving as anycast default gateway for selected subnet. In case of multi-site V-Net, multi-site subnet should be configured under Subnets section.
[6] | sites[n].switchPorts                   | []          | List of switchPorts.
[7] | sites[n].switchPorts[n].name           | ""          | SwitchPorts name.
[8] | sites[n].switchPorts[n].vlanId         | nil         | VLAN tag for current port. If `vlanid` is not set - means port untagged
[9] | sites[n].switchPorts[n].state          | active      | Port state. Allowed values: `active` or `disable`. 
