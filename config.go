package main

import (
  "encoding/json"
  "io/ioutil"
)


type CfgNs struct {
  Name string `json:"name"`
  Type string `json:"type"`
  ContainerId string `json:"container-id"`
  Pid string  `json:"pid"`
}

type CfgInterface struct {
  Name string `json:"name"`
  Namespace string `json:"namespace"`
  Type string `json:"type"`

  PeerName string `json:"peer_name"`
  PeerNamespace string `json:"peer_namespace"`

  Slave []string `json:"slave"`
}

type CfgIp struct {
  Interface string `json:"interface"`
  Namespace string `json:"namespace"`
  Address string `json:"address"`
}

type Cfg struct {
  Namespaces []CfgNs `json:"namespaces"`
  Interfaces []CfgInterface `json:"interfaces"`
  Ips []CfgIp `json:"ips"`
}


func configRead(fname string) Cfg {
  data, err := ioutil.ReadFile(fname)
  if err != nil {
    panic(err)
  }
  var cfg Cfg
  err = json.Unmarshal(data, &cfg)
  if err != nil {
    panic(err)
  }
  return cfg
}