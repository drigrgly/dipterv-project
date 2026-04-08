# 2026. 04. 08. - Premium STUNner and turn offloading

After setting up the premium tier STUNner following [`the guide`](https://docs.l7mp.io/en/stable/PREMIUM_INSTALL/), I used the simple-tunnel example, to test whether setting up offloading was successful or not:


## Without offloading
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