# nsutil
## introduction
Simple utility to simplify creation of network namespaces, interfaces and bridges.
Purpose is to create "virtualized" network environment for testing purposes.

Required topology is described in a simple json file.
Following example creates 3 network namespaces:
- ns1 - with network interface eth0 (which is veth with other end inside namespace nsb)
- ns2 - with network interface eth0 (which is veth with other end inside namespace nsb)
- nsb - with network interface b0 (which is bridge connecting veth interfaces from ns1 and ns2)
```
{
  "namespaces": [
    {"name": "ns1"},
    {"name": "ns2"},
    {"name": "nsb"}
  ],

  "interfaces": [
    {"namespace": "ns1", "name": "eth0", "type": "veth", "peer_namespace": "nsb", "peer_name": "c1"},
    {"namespace": "ns2", "name": "eth0", "type": "veth", "peer_namespace": "nsb", "peer_name": "c2"},
    {"namespace": "nsb", "name": "b0", "type": "bridge", "slave": ["c1", "c2"]}
  ],
 
  "ips": [
    {"namespace": "ns1", "interface": "eth0", "address": "1.1.1.1/24"},
    {"namespace": "ns1", "interface": "eth0", "address": "fd00::100/64"},
    {"namespace": "ns2", "interface": "eth0", "address": "1.1.1.2/24"},
    {"namespace": "ns2", "interface": "eth0", "address": "fd00::101/64"}
  ],
  "exec": [
    {"namespace": "ns1", "command": "ip route add 10.0.1.0/24 via 10.0.0.20"}
  ]
}
```

```
+-------   +--------+   +------+
|  ns1 |   |   nsb  |   |  ns2 |
|      |   |        |   |      |
|  eth0-----c1-b0-c2-----eth0  |
+------+   +--------+   +------+
```

## compile
To compile utility just clone repository and run `go build`.

## create
Store example configuration to config.json and run: `./nsutil create`.
You can then check what is inside network namespaces and try to ping other ends:
```
ip netns exec ns1 ip addr
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
3: eth0@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default 
    link/ether 2a:25:6b:c9:51:86 brd ff:ff:ff:ff:ff:ff link-netns nsb
    inet 1.1.1.1/24 brd 1.1.1.255 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fd00::100/64 scope global 
       valid_lft forever preferred_lft forever
    inet6 fe80::2825:6bff:fec9:5186/64 scope link 
       valid_lft forever preferred_lft forever

``` 

```
ip netns exec ns1 ping fd00::101
PING fd00::101(fd00::101) 56 data bytes
64 bytes from fd00::101: icmp_seq=1 ttl=64 time=0.115 ms

ip netns exec ns1 ping 1.1.1.2  
PING 1.1.1.2 (1.1.1.2) 56(84) bytes of data.
64 bytes from 1.1.1.2: icmp_seq=2 ttl=64 time=0.069 ms

```

## delete
To cleanup resources created by nsutil run: `nsutil delete`.
Utility will attempt to delete all resources it did create.

## config file

### namespaces
- In basic form new named namespace is created: `{"name": "ns1"}`
- It is also possible to use existing namespace of existing container using this syntax: `{"name": "ns3", "type": "container", "container-id": "3c521aadf445"}`
- To use existing namespace attached to known pid use: `{"name": "ns4", "type": "pid", "pid": "4144225"}`

### interfaces
supported interfaces are:

#### veth
To create veth interface use: `{"namespace": "ns1", "name": "eth9", "type": "veth", "peer_namespace": "nsb", "peer_name": "c1"}`.
This example will create veth pair with one end in namespace "ns1" and interface named "eth2". Other end will be in namespace "nsb" with interface named "c1".

#### bridge
To create bridge use for example `{"namespace": "nsb", "name": "b0", "type": "bridge", "slave": ["c1", "c2", "c3"]}`.
This example will create bridge named b0 in namespace nsb. Interfaces c1, c2, c3 will be attached to this bridge.

#### macvlan
To create macvlan interface use for example `{"namespace": "ns1", "name": "eth0", "type": "macvlan", "parent": "eth0"}`.
This example will create macvlan interface named eth0 in namespace ns1. Parent interface is eth0 from root namespace.

#### ipvlan
To create ipvlan interface use for example `{"namespace": "ns1", "name": "eth0", "type": "ipvlan", "parent": "eth0"}`.
This example will create ipvlan interface named eth0 in namespace ns1. Parent interface is eth0 from root namespace.

#### vlan
To create vlan interface based on existing interface use for example `{"namespace": "ns1", "name": "eth0", "type": "vlan", "vlan_id": 100}`.
This example will create vlan interface named eth0-100 in namespace ns1 based don eth0.