# Note
Register the Netris [CRDs](https://github.com/netrisai/netris-operator/tree/master/deploy) in the Kubernetes cluster before creating objects.


### Site Attributes
```
apiVersion: k8s.netris.ai/v1alpha1
kind: Site
metadata:
  name: santa-clara
spec:
  publicAsn: 65001                                    # [1]
  rohAsn: 65502                                       # [2]
  vmAsn: 65503                                        # [3]
  rohRoutingProfile: default                          # [4]
  siteMesh: hub                                       # [5]
  aclDefaultPolicy: permit                            # [6]
```

Ref | Attribute                              | Default     | Description
----| -------------------------------------- | ----------- | ----------------
[1] | publicAsn                              | nil         | Site public ASN that should be used for external bgp peer configuration.
[2] | rohAsn                                 | nil         | ASN for ROH (Routing on the Host) compute instances, should be unique within the scope of a site, can be same for different sites.
[3] | vmAsn                                  | nil         | ASN for ROH (Routing on the Host) virtual compute instances, should be unique within the scope of a site, can be same for different sites.
[4] | rohRoutingProfile                      | ""          | ROH Routing profile defines set of routing prefixes to be advertised to ROH instances. Possible values: `default`, `default_agg`, `full`. Default route only - Will advertise 0.0.0.0/0 + loopback address of physically connected switch. Default + Aggregate - Will add prefixes of defined subnets + "Default" profile. Full - Will advertise all prefixes available in the routing table of the connected switch.
[5] | siteMesh                               | ""          | Site to site VPN mode. Possible values: `disabled`, `hub`, `spoke`, `dynamicSpoke`. 
[6] | aclDefaultPolicy                       | ""          | Possible values: `permit` or `deny`. Deny - Layer-3 packet forwarding is denied by default. ACLs are required to permit necessary traffic flows. Deny ACLs will be applied before Permit ACLs. Permit - Layer-3 packet forwarding is allowed by default. ACLs are required to deny unwanted traffic flows. Permit ACLs will be applied before Deny ACLs.


### Allocation Attributes
```
apiVersion: k8s.netris.ai/v1alpha1
kind: Allocation
metadata:
  name: my-allocation
spec:
  prefix: 192.0.2.0/24                                   # [1]
  tenant: Admin                                          # [2]
```

Ref | Attribute                              | Default     | Description
----| -------------------------------------- | ----------- | ----------------
[1] | prefix                                 | ""          | Allocation ipv4/ipv6 prefix.
[2] | tenant                                 | ""          | Users of this tenant will be permitted to manage subnets under this allocation.


### Subnet Attributes
```
apiVersion: k8s.netris.ai/v1alpha1
kind: Subnet
metadata:
  name: my-subnet
spec:
  prefix: 192.0.2.0/24                                   # [1]
  tenant: Admin                                          # [2]
  purpose: management                                    # [3]
  defaultGateway: 192.0.2.1                              # [4] optional
  sites:                                                 # [5]
  - santa-clara
```

Ref | Attribute                              | Default     | Description
----| -------------------------------------- | ----------- | ----------------
[1] | prefix                                 | ""          | Subnet ipv4/ipv6 prefix.
[2] | tenant                                 | ""          | Users of this tenant will be permitted to edit this subnet.
[3] | purpose                                | ""          | Describes which kind of service will be able to use this subnet. Possible values: `common`, `loopback`, `management`, `load-balancer`, `nat`, `inactive`.
[4] | defaultGateway                         | ""          | Optional. Use when purpose is set to `management`.
[5] | sites                                  | []          | List of sites where this subnet is available.


### Switch Attributes
```
apiVersion: k8s.netris.ai/v1alpha1
kind: Switch
metadata:
  name: my-sw01
spec:
  tenant: Admin                                          # [1]
  description: My Switch01                               # [2] optional
  nos: cumulus_linux                                     # [3]
  site: santa-clara                                      # [4]
  asn: 4200000021                                        # [5] optional
  profile: my-profile                                    # [6] optional
  mainIp: 198.51.100.3                                   # [7] optional
  mgmtIp: 192.0.2.3                                      # [8] optional
  portsCount: 16                                         # [9]
```

Ref | Attribute                              | Default       | Description
----| -------------------------------------- | ------------- | ----------------
[1] | tenant                                 | ""            | Users of this tenant will be permitted to edit this unit.
[2] | description                            | ""            | Optional. Switch description.
[3] | nos                                    | ""            | Switch OS. Possible values: `cumulus_linux`, `sonic`, `ubuntu_switch_dev`.
[4] | site                                   | ""            | The site where this device belongs.
[5] | asn                                    | automatically | Optional. Switch AS numbers. If `asn` key isn't set, the controller will assign automatically from `System ASN range`.
[6] | profile                                | ""            | Optional. An inventory profile name to define global configuration (NTP, DNS, timezone, etc…).
[7] | mainIp                                 | automatically | Optional. A unique IP address which will be used as a loopback address of this unit. If `mainIp` key isn't set the controller will assign automatically from subnets with relevant purpose.
[8] | mgmtIp                                 | automatically | Optional. A unique IP address to be used on out of band management interface. If `mgmtIp` key isn't set the controller will assign automatically from subnets with relevant purpose.
[9] | portsCount                             | nil           | Preliminary port count is used for definition of topology. Possible values: `16`, `32`, `48`, `54`, `56`.


### Softgate Attributes
```
apiVersion: k8s.netris.ai/v1alpha1
kind: Softgate
metadata:
  name: my-sg01
spec:
  tenant: Admin                                          # [1]
  description: My Softgate01                             # [2] optional
  site: santa-clara                                      # [3]
  profile: my-profile                                    # [4] optional
  mainIp: 198.51.100.2                                   # [5] optional
  mgmtIp: 192.0.2.2                                      # [6] optional
```

Ref | Attribute                              | Default       | Description
----| -------------------------------------- | ------------- | ----------------
[1] | tenant                                 | ""            | Users of this tenant will be permitted to edit this unit.
[2] | description                            | ""            | Optional. Softgate description.
[3] | site                                   | ""            | The site where this device belongs.
[4] | profile                                | ""            | Optional. An inventory profile name to define global configuration (NTP, DNS, timezone, etc…).
[5] | mainIp                                 | automatically | Optional. A unique IP address which will be used as a loopback address of this unit. If `mainIp` key isn't set the controller will assign automatically from subnets with relevant purpose.
[6] | mgmtIp                                 | automatically | Optional. A unique IP address to be used on out of band management interface. If `mgmtIp` key isn't set the controller will assign automatically from subnets with relevant purpose.


### Controller Attributes
```
apiVersion: k8s.netris.ai/v1alpha1
kind: Controller
metadata:
  name: my-controller
spec:
  tenant: Admin                                          # [1]
  description: My Controller                             # [2] optional
  site: santa-clara                                      # [3]
  mainIp: 198.51.100.10                                  # [4] optional
```

Ref | Attribute                              | Default       | Description
----| -------------------------------------- | ------------- | ----------------
[1] | tenant                                 | ""            | Users of this tenant will be permitted to edit this unit.
[2] | description                            | ""            | Optional. Controller description.
[3] | site                                   | ""            | The site where this controller belongs.
[4] | mainIp                                 | automatically | Optional. A unique IP address which will be used as a loopback address of this unit. If `mainIp` key isn't set the controller will assign automatically from subnets with relevant purpose.


### VNet Attributes

```
apiVersion: k8s.netris.ai/v1alpha1
kind: VNet
metadata:
  name: my-vnet
spec:
  ownerTenant: Admin                                     # [1]
  guestTenants: []                                       # [2]
  state: active                                          # [3] optional
  sites:                                                 # [4]
    - name: santa-clara                                  # [5]
      gateways:                                          # [6]
        - 192.0.2.1/24
        - 2001:db8:acad::fffe/64
      switchPorts:                                       # [7]
        - name: swp4@rlab-leaf1                          # [8]
          vlanId: 1050                                   # [9] optional
        - name: swp7@rlab-leaf1
          state: disable                                 # [10] optional
```

Ref | Attribute                              | Default     | Description
----| -------------------------------------- | ----------- | ----------------
[1] | ownerTenant                            | ""          | Users with permission to owner tenant can manage parameters of the V-Net as well as add/edit/remove ports assigned to any of tenants where user has permission.
[2] | guestTenants                           | []          | List of tenants allowed to add/edit/remove ports to the V-Net but not allowed to manage other parameters of the circuit.
[3] | state                                  | active      | V-Net state. Allowed values: `active` or `disable`. 
[4] | sites                                  | []          | List of sites. Ports from these sites will be allowed to participate to the V-Net. Multi-site circuits are possible for sites connected through a backbone port.
[5] | sites[n].name                          | ""          | Site's name.
[6] | sites[n].gateways                      | []          | List of gateways. Selected address will be serving as anycast default gateway for selected subnet. In case of multi-site V-Net, multi-site subnet should be configured under Subnets section.
[7] | sites[n].switchPorts                   | []          | List of switchPorts.
[8] | sites[n].switchPorts[n].name           | ""          | SwitchPorts name.
[9] | sites[n].switchPorts[n].vlanId         | nil         | VLAN tag for current port. If `vlanid` is not set - means port untagged
[10] | sites[n].switchPorts[n].state         | active      | Port state. Allowed values: `active` or `disable`. 


### BGP Attributes

```
apiVersion: k8s.netris.ai/v1alpha1
kind: BGP
metadata:
  name: my-bgp
spec:
  site: santa-clara                                  # [1]
  hardware: softgate1                                # [2] Ignoring when transport.type == vnet
  neighborAs: 23456                                  # [3]
  transport:                                         # [4]
    type: port                                       # [5] optional
    name: swp5@rlab-spine1                           # [6]   
    vlanId: 4                                        # [7] optional. Ignoring when transport.type == vnet
  localIP: 172.16.0.1/30                             # [8]
  remoteIP: 172.16.0.2/30                            # [9]
  description: someDesc                              # [10] optional
  state: enabled                                     # [11] optional
  multihop:                                          # [12] optional
    neighborAddress: 8.8.8.8                         # [13] optional
    updateSource: 10.254.97.33                       # [14] optional
    hops: 5                                          # [15] optional
  bgpPassword: somestrongpass                        # [16] optional
  allowAsIn: 5                                       # [17] optional
  defaultOriginate: false                            # [18] optional
  prefixInboundMax: 10000                            # [19] optional
  inboundRouteMap: my-in-rm                          # [20] optional
  outboundRouteMap: my-out-rm                        # [21] optional
  localPreference: 100                               # [22] optional. Ignoring when *RouteMap defined
  weight: 0                                          # [23] optional. Ignoring when *RouteMap defined
  prependInbound: 2                                  # [24] optional. Ignoring when *RouteMap defined
  prependOutbound: 1                                 # [25] optional. Ignoring when *RouteMap defined
  prefixListInbound:                                 # [26] optional. Ignoring when *RouteMap defined
    - deny 127.0.0.0/8 le 32
    - permit 0.0.0.0/0 le 24
  prefixListOutbound:                                # [27] optional. Ignoring when *RouteMap defined
    - permit 192.0.2.0/24
  sendBGPCommunity:                                  # [28] optional. Ignoring when *RouteMap defined
    - 65501:777
    - 65501:779
```

Ref | Attribute                              | Default     | Description
----| -------------------------------------- | ----------- | ----------------
[1] | sites                                  | ""          | BGP session site
[2] | hardware                               | "auto"      | Defines hardware for Layer-3 and BGP session termination. Ignoring when transport.type == vnet
[3] | neighborAs                             | 0           | BGP neighbor AS number
[4] | transport                              | {}          | Physical port where BGP neighbor cable is connected or an existing V-Net service
[5] | transport.type                         | port        | Possible values: `port` or `vnet`
[6] | transport.name                         | ""          | Possible values: `portName@switchName` or `vnetName`
[7] | transport.vlanId                       | nil         | Ignoring when transport.type == vnet
[8] | localIP                                | ""          | BGP session local ip
[9] | remoteIP                               | ""          | BGP session remote ip
[10]| description                            | ""          | BGP session description
[11]| state                                  | enabled     | Possible values: `enabled` or `disabled`; enabled - initiating and waiting for BGP connections, disabled - disable Layer-2 tunnel and Layer-3 address.
[12]| multihop                               | {}          | Multihop BGP session configurations
[13]| multihop.neighborAddress               | ""          | -
[14]| multihop.updateSource                  | ""          | -
[15]| multihop.hops                          | 0           | -
[16]| bgpPassword                            | ""          | BGP session password
[17]| allowAsIn                              | 0           | Optionally allow number of occurrences of the own AS number in received prefix AS-path.
[18]| defaultOriginate                       | false       | Originate default route to current neighbor.
[19]| prefixInboundMax                       | 0           | BGP session will be terminated if neighbor advertises more prefixes than defined.
[20]| inboundRouteMap                        | ""          | Reference to route-map resource.
[21]| outboundRouteMap                       | ""          | Reference to route-map resource. 
[22]| localPreference                        | 100         | -
[23]| weight                                 | 0           | -
[24]| prependInbound                         | 0           | Number of times to prepend self AS to as-path of received prefix advertisements.
[25]| prependOutbound                        | 0           | Number of times to prepend self AS to as-path being advertised to neighbors.
[26]| prefixListInbound                      | []          | -
[27]| prefixListOutbound                     | []          | Define outbound prefix list, if not defined autogenerated prefix list will apply which will permit defined allocations and assignments, and will deny all private addresses.
[28]| sendBGPCommunity                       | []          | Send BGP Community Unconditionally advertise defined list of BGP communities towards BGP neighbor. Format: AA:NN Community number in AA:NN format (where AA and NN are (0-65535)) or local-AS|no-advertise|no-export|internet or additive


### L4LB Attributes

```
apiVersion: k8s.netris.ai/v1alpha1
kind: L4LB
metadata:
  name: my-l4lb
spec:
  ownerTenant: Admin                                 # [1]
  site: santa-clara                                  # [2]           
  state: active                                      # [3]  optional
  protocol: tcp                                      # [4]  optional
  frontend:
    port: 31434                                      # [5]
    ip: 203.0.113.7                                  # [6]  optional
  backend:                                           # [7]
    - 192.0.2.100:443
    - 192.0.2.101:443
  check:                                             # [8]  optional. Ignoring when protocol == udp
    type: http                                       # [9] optional
    timeout: 3000                                    # [10] optional
    requestPath: /                                   # [11] optional. Ignoring when check.type == tcp
```

Ref | Attribute                              | Default                | Description
----| -------------------------------------- | -----------------------| ----------------
[1] | ownerTenant                            | ""                     | Users of this Tenant will be permitted to edit this service
[2] | site                                   | ""                     | Resources defined in the selected site will be permitted to be used as backed entries for this L4 Load Balancer service
[3] | state                                  | active                 | Administrative status. Possible values: `active` or `disable`
[4] | protocol                               | tcp                    | Protocol. Possible values: `tcp` or `udp`
[5] | frontend.port                          | nil                    | L4LB frontend port
[6] | frontend.ip                            | *Assign Automatically* | L4LB frontend ip
[7] | backend                                | []                     | List of backend servers. Possible values: ip:port
[8] | check                                  | {}                     | A health check determines whether instances in the target pool are healthy. If protocol == `udp` then check.type will be `none`
[9]| check.type                              | tcp                    | Probe type. Possible values: `tcp`, `http` or `none`
[10]| check.timeout                          | 2000                   | Probe timeout
[11]| check.requestPath                      | /                      | Http probe path. Ignoring when check.type == tcp


# Annotations

> Annotation keys and values can only be strings. Other types, such as boolean or numeric values must be quoted, i.e. "true", "false", "100".


Name                                   | Default      |Values              | Description
-------------------------------------- | ------------ | ------------------ | ----------------
`resource.k8s.netris.ai/import`        | "false"      |"true" or "false"   | Allow importing existing resources. 
`resource.k8s.netris.ai/reclaimPolicy` | "delete"     |"retain" or "delete"| Resources reclaim policy.


# Calico Integration

Calico nodes exchange routing information over BGP to enable reachability for Calico networked workloads. Netris can also integrate with your Calico CNI. It will create BGP peers with your cluster's nodes, then will disable Calico Node to Node mesh. For more details, get familiar with [calico docs](https://docs.projectcalico.org/networking/bgp).

Add this annotation to enable Netris-Calico Integration.
```
kubectl annotate bgpconfigurations default manage.k8s.netris.ai/calico='true'
```
