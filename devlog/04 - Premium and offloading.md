# 2026. 04. 08. - Premium STUNner and turn offloading

After setting up the premium tier STUNner following [`the guide`](https://docs.l7mp.io/en/stable/PREMIUM_INSTALL/), I used the simple-tunnel example, to test whether setting up offloading was successful or not:


## Without offloading
```bash
stunnerctl -n stunner status udp-gateway
```
```
stunner/udp-gateway-56554cb6cd-llg5k:
        admin:{id="stunner/udp-gateway",logLevel="all:INFO",health-check="http://:8086",quota=0,license-info={tier=enterprise,unlocked-features=[UserQuota,RelayAddressDiscovery,STUNServer,DaemonSet,TURNOffload],valid-until=2026-04-08 21:00:33 +0000 UTC},offload=Auto[all]}
        static-auth:{realm="stunner.l7mp.io",username="<SECRET>",password="<SECRET>"}
        listeners:"stunner/udp-gateway/udp-listener":{turn://10.42.0.27:3478?transport=TURN-UDP,public=152.66.181.21:3478,cert/key=-/-,routes=[stunner/iperf-server]},offload(rx/tx): 0/0 pkts 0/0 bytes
        clusters:"stunner/iperf-server":{type="STATIC",protocol="UDP",endpoints=[10.42.0.10,10.43.144.0]},offload(rx/tx): 0/0 pkts 0/0 bytes
        allocs:1/status=TERMINATING
```
## With offloading
```
stunner/udp-gateway-764496ddc9-v9mbk:
        admin:{id="stunner/udp-gateway",logLevel="all:INFO",health-check="http://:8086",quota=0,license-info={tier=enterprise,unlocked-features=[UserQuota,RelayAddressDiscovery,STUNServer,DaemonSet,TURNOffload],valid-until=2026-04-08 21:00:35 +0000 UTC},offload=Auto[all]}
        static-auth:{realm="stunner.l7mp.io",username="<SECRET>",password="<SECRET>"}
        listeners:"stunner/udp-gateway/udp-listener":{turn://10.42.0.28:3478?transport=TURN-UDP,public=152.66.181.21:3478,cert/key=-/-,routes=[stunner/iperf-server]},offload(rx/tx): 9999/1 pkts 1459854/174 bytes
        clusters:"stunner/iperf-server":{type="STATIC",protocol="UDP",endpoints=[10.42.0.10,10.43.144.0]},offload(rx/tx): 1/9999 pkts 170/1419858 bytes
        allocs:1/status=READY
 ```

We can see the main difference at the `offload(rx/tx)`, when offloading is successfully configured, the offloaded packet statistics can be read from the output

## Further verifying offloading

For further checks, firstly I installed [`bpftool`](https://github.com/libbpf/bpftool).

The first step was to see what eBPF maps are present:
```bash
sudo bpftool map show
```

From this list, those, that are important to us are the following:
```bash
52: lru_hash  name stunner_downstr  flags 0x0
        key 8B  value 16B  max_entries 10240  memlock 1000704B
        btf_id 252
        pids stunnerd(7772)
53: lru_hash  name stunner_upstrea  flags 0x0
        key 12B  value 12B  max_entries 10240  memlock 1082624B
        btf_id 253
        pids stunnerd(7772)
54: lru_hash  name stunner_stats_m  flags 0x0
        key 4B  value 24B  max_entries 10240  memlock 1082624B
        btf_id 254
        pids stunnerd(7772)
```

Using the `watch` bash utility, we can check whether the values change over time, as we run the simple-tunnel test

**Example**


```bash
watch -n1 "sudo bpftool map dump id 54"
```

```bash
[{
        "key": {
            "name_hash": 22129,
            "flags": 3
        },
        "value": {
            "pkts": 79993,
            "bytes": 11678978,
            "timestamp_last": 2879213701963
        }
    },{
        "key": {
            "name_hash": 64472,
            "flags": 0
        },
        "value": {
            "pkts": 79993,
            "bytes": 11359006,
            "timestamp_last": 2879213703429
        }
    }
.
.
.
```

```bash
[{
        "key": {
            "name_hash": 22129,
            "flags": 3
        },
        "value": {
            "pkts": 88979,
            "bytes": 12990934,
            "timestamp_last": 3527905801652
        }
    },{
        "key": {
            "name_hash": 64472,
            "flags": 0
        },
        "value": {
            "pkts": 88979,
            "bytes": 12635018,
            "timestamp_last": 3527905804027
        }
    },
.
.
.
```
