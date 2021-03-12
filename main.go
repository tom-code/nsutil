package main

import (
  "fmt"
  "os/exec"
  "runtime"
  "strconv"

  "github.com/containernetworking/plugins/pkg/ns"
  "github.com/spf13/cobra"
  "github.com/vishvananda/netlink"
  "github.com/vishvananda/netns"
)

func make_veth(ns1, ns2 ns.NetNS, name1, name2 string, queues int) {
  veth := netlink.Veth {
    LinkAttrs: netlink.LinkAttrs {
      //MTU: 1000,
      Name: name1,
      NumTxQueues: queues,
      NumRxQueues: queues,
      //Namespace: netlink.NsFd(int(netns.Fd())),
    },
    PeerName: "tmp",
  }
  err := ns1.Do(func(_ ns.NetNS) error {

    err := netlink.LinkAdd(&veth)
    if err != nil {
      return err
    }
    netlink.LinkSetUp(&veth)
    veth2, err := netlink.LinkByName("tmp")
    if err != nil {
      return err
    }
    err = netlink.LinkSetNsFd(veth2, int(ns2.Fd()))
    if err != nil {
      return err
    }
    return nil
  })
  if err != nil {
    fmt.Printf("can't create veth %s/%s %s/%s %s\n", ns1.Path(), name1, ns2.Path(), name2, err)
    return
  }

  ns2.Do(func(_ ns.NetNS) error {
    veth2, err := netlink.LinkByName("tmp")
    if err != nil {
      panic(err)
    }
    netlink.LinkSetName(veth2, name2)
    netlink.LinkSetUp(veth2)
    return nil
  })
}

func make_macvlan(ns1 ns.NetNS, name, parent string) {
  rootnetns, err := ns.GetNS("/proc/1/ns/net")
  if err != nil {
    panic(err)
  }
  defer rootnetns.Close()
  err = rootnetns.Do(func(_ ns.NetNS) error {
    parent, err := netlink.LinkByName(parent)
    if err != nil {
      return err
    }
    mv := netlink.Macvlan {
      LinkAttrs: netlink.LinkAttrs {
        MTU: 1000,
        Name: name,
        ParentIndex: parent.Attrs().Index,
        Namespace: netlink.NsFd(int(ns1.Fd())),
      },
    }
    err = netlink.LinkAdd(&mv)
    if err != nil {
      return err
    }
    return nil
  })
  if err != nil {
    fmt.Println(err)
  }
}

func make_ipvlan(ns1 ns.NetNS, name, parent string) {
  rootnetns, err := ns.GetNS("/proc/1/ns/net")
  if err != nil {
    panic(err)
  }
  defer rootnetns.Close()
  err = rootnetns.Do(func(_ ns.NetNS) error {
    parent, err := netlink.LinkByName(parent)
    if err != nil {
      return err
    }
    mv := netlink.IPVlan {
      LinkAttrs: netlink.LinkAttrs {
        MTU: 1001,
        Name: name,
        ParentIndex: parent.Attrs().Index,
        Namespace: netlink.NsFd(int(ns1.Fd())),
      },
      Mode: netlink.IPVLAN_MODE_L2,
    }
    err = netlink.LinkAdd(&mv)
    if err != nil {
      return err
    }
    return nil
  })
  if err != nil {
    fmt.Println(err)
  }
}

func make_bridge(ns1 ns.NetNS, name string) {
  br := netlink.Bridge {
    LinkAttrs: netlink.LinkAttrs{
      Name: name,
    },
  }
  ns1.Do(func(_ ns.NetNS) error {
    netlink.LinkAdd(&br)
    netlink.LinkSetUp(&br)
    return nil
  })
}

func make_vlan(ns1 ns.NetNS, iface string, id int) {
  parent_link := -1
  ns1.Do(func(_ ns.NetNS) error {
    parent_linkl, err := netlink.LinkByName(iface)
    parent_link = parent_linkl.Attrs().Index
    return err
  })
  vlan := netlink.Vlan {
    LinkAttrs: netlink.LinkAttrs{
      Name: iface+"-"+strconv.Itoa(id),
      ParentIndex: parent_link,
    },
    VlanId: id,
  }
  ns1.Do(func(_ ns.NetNS) error {
    netlink.LinkAdd(&vlan)
    netlink.LinkSetUp(&vlan)
    return nil
  })
}

func link_del(ns1 ns.NetNS, name string) {
  ns1.Do(func(_ ns.NetNS) error {
    l, err := netlink.LinkByName(name)
    if err != nil {
      fmt.Printf("can't find link %s %s\n", name, err)
      return err
    }
    netlink.LinkDel(l)
    return nil
  })
}

func make_ns(name string) ns.NetNS {
  nsn, err := netns.NewNamed(name)
  if err != nil {
    fmt.Printf("can't create named ns %s %s\n", name, err)
  }
  defer nsn.Close()

  netns1, err := ns.GetNS("/run/netns/"+name)  //var/run/netns ??
  if err != nil {
    panic(err)
  }

  err = netns1.Do(func(_ ns.NetNS) error {
    lo, err := netlink.LinkByName("lo")
    if err == nil {
      netlink.LinkSetUp(lo)
    }
    return nil
  })
  return netns1
}

func add_ip(namespace ns.NetNS, iface string, ip string) {
  namespace.Do(func(_ ns.NetNS) error {
    link, err := netlink.LinkByName(iface)
    if err != nil {
      fmt.Printf("can't get link %s %s\n", iface, err)
      return err
    }
    a, err := netlink.ParseAddr(ip)
    if err != nil {
      fmt.Printf("can't parse address %s %s\n", ip, err)
      return err
    }
    err = netlink.AddrAdd(link, a)
    if err != nil {
      fmt.Printf("can't add address %s %s %s\n", iface, ip, err)
      return err
    }
    return nil
  })
}

func add_to_bridge(namespace ns.NetNS, iface string, bridge string) {
  namespace.Do(func(_ ns.NetNS) error {
    iface, err := netlink.LinkByName(iface)
    if err != nil {
      fmt.Printf("can't get link %s %s\n", iface, err)
      return err
    }
    ifacebr, err := netlink.LinkByName(bridge)
    if err != nil {
      fmt.Printf("can't get bridge %s %s\n", bridge, err)
      return err
    }
    br := ifacebr.(*netlink.Bridge)
    netlink.LinkSetMaster(iface, br)
    return nil
  })
}


func create(cfg Cfg) {
  runtime.LockOSThread()
  defer runtime.UnlockOSThread()

  nsmap := make(map[string]ns.NetNS)
  for _, nsi := range(cfg.Namespaces) {
    //update get_nsmap when adding new types !
    if nsi.Type == "container" {
      nsh, err := netns.GetFromDocker(nsi.ContainerId)
      if err != nil {
        panic(err)
      }
      netns.Set(nsh)
      nsmap[nsi.Name], _ = ns.GetCurrentNS()
    }
    if nsi.Type == "pid" {
      pid, err := strconv.Atoi(nsi.Pid)
      if err != nil {
        panic(err)
      }
      nsh, err := netns.GetFromPid(pid)
      if err != nil {
        panic(err)
      }
      netns.Set(nsh)
      nsmap[nsi.Name], _ = ns.GetCurrentNS()
    }
    if nsi.Type == "" {
      nsmap[nsi.Name] = make_ns(nsi.Name)
    }
  }

  for _, iface := range(cfg.Interfaces) {
    if iface.Type == "veth" {
      ns1 := nsmap[iface.Namespace]
      ns2 := nsmap[iface.PeerNamespace]
      make_veth(ns1, ns2, iface.Name, iface.PeerName, iface.Queues)
    }
    if iface.Type == "macvlan" {
      ns1, ok := nsmap[iface.Namespace]
      if !ok {
        panic("")
      }
      make_macvlan(ns1, iface.Name, iface.Parent)
    }
    if iface.Type == "ipvlan" {
      ns1, ok := nsmap[iface.Namespace]
      if !ok {
        panic("")
      }
      make_ipvlan(ns1, iface.Name, iface.Parent)
    }
    if iface.Type == "bridge" {
      ns1 := nsmap[iface.Namespace]
      make_bridge(ns1, iface.Name)
    }
    if iface.Type == "vlan" {
      ns1 := nsmap[iface.Namespace]
      make_vlan(ns1, iface.Name, iface.VlanId)
    }
  }
  for _, iface := range(cfg.Interfaces) {
    if iface.Type == "bridge" {
      for _, slave := range(iface.Slave) {
        ns1 := nsmap[iface.Namespace]
        add_to_bridge(ns1, slave, iface.Name)
      }
    }
  }
  for _, ip := range(cfg.Ips) {
    ns1 := nsmap[ip.Namespace]
    add_ip(ns1, ip.Interface, ip.Address)
  }

  for _, exe := range(cfg.Execs) {
    ns1 := nsmap[exe.Namespace]
    cmd := exec.Command("sh", "-c", exe.Command)
    ns1.Do(func(_ ns.NetNS) error {
      stdoutStderr, err := cmd.CombinedOutput()
      if err != nil {
        fmt.Println(err)
        return err
      }
      if len(stdoutStderr) > 0 {
        fmt.Printf("%s\n", stdoutStderr)
      }
      return nil
    })
  }
}

func get_nsmap(cfg Cfg) map[string]ns.NetNS {
  nsmap := make(map[string]ns.NetNS)
  for _, nsi := range(cfg.Namespaces) {
    if nsi.Type == "container" {
      nsh, err := netns.GetFromDocker(nsi.ContainerId)
      if err != nil {
        fmt.Printf("can't get ns container:%s %s\n", nsi.ContainerId, err)
        continue
      }
      netns.Set(nsh)
      nsmap[nsi.Name], _ = ns.GetCurrentNS()
    }
    if nsi.Type == "pid" {
      pid, err := strconv.Atoi(nsi.Pid)
      if err != nil {
        fmt.Println(err)
        continue
      }
      nsh, err := netns.GetFromPid(pid)
      if err != nil {
        fmt.Printf("can't get ns pid:%s %s\n", nsi.Pid, err)
        continue
      }
      netns.Set(nsh)
      nsmap[nsi.Name], _ = ns.GetCurrentNS()
    }
    if nsi.Type == "" {
      nsh, err := netns.GetFromName(nsi.Name)
      if err != nil {
        fmt.Printf("can't get ns name:%s %s\n", nsi.Name, err)
        continue
      }
      netns.Set(nsh)
      nsmap[nsi.Name], _ = ns.GetCurrentNS()
    }
  }
  return nsmap
}

func delete(cfg Cfg) {
  runtime.LockOSThread()
  defer runtime.UnlockOSThread()
  nsmap := get_nsmap(cfg)

  for _, lnk := range(cfg.Interfaces) {
    ns1, ok := nsmap[lnk.Namespace]
    if !ok {
      fmt.Printf("can't get ns %s\n", lnk.Namespace)
      continue
    }
    link_del(ns1, lnk.Name)
  }
  for _, ns := range(cfg.Namespaces) {
    if ns.Type == "" {
      netns.DeleteNamed(ns.Name)
    }
  }

}

func main() {

  rootCmd := &cobra.Command{
		Use:   "nsutil",
		Short: "Utility to create netns topology",
  }
  rootCmd.PersistentFlags().StringP("config", "c", "config.json", "configuration file")
  createCmd := &cobra.Command{
    Use: "create",
    Run: func(cmd *cobra.Command, args []string) {
      cn, _ := cmd.Flags().GetString("config")
      cfg := configRead(cn)
      create(cfg)
    },
  }
  rootCmd.AddCommand(createCmd)

  deleteCmd := &cobra.Command{
    Use: "delete",
    Run: func(cmd *cobra.Command, args []string) {
      cn, _ := cmd.Flags().GetString("config")
      cfg := configRead(cn)
      delete(cfg)
    },
  }
  rootCmd.AddCommand(deleteCmd)
  
  rootCmd.Execute()
}

