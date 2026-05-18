# FillARPd Fill in /32s to the Route Table from Arp Snooping and Sweeping

This tool/daemon is intended to be used for Proxy Arp Setups (on the proxy arp gateway) 
where you have to see all the devices behind a specific interface even when your proxy arp host
might not necessarily be on that subnet (e.g your proxy arp host is /32 sharing an IP)
This tool allows you to scan for those devices and as long as they can make ARP requests back to your source ip it will fill in the route table with /32s with the devices it discovers.
It also keeps the arp tables up to date when devices are no longer up, allowing the IPs to be reallocated without conflict.

# Context

fillarpd discovers devices in 2 ways, ARP Snooping and sweeping. 
- ARP Snooping is reactive and will update within 5 milliseconds of a device making an ARP request
allowing the node on the other side of Proxy Arp to send it's data back without
noticing the route table changes.
- Sweeping is proactive, every sweep interval and at startup fillarpd will lookup all the ips in the subnet to ensure that IPs that haven't checked in are active. If they are not active their route
will be removed. The shorter the Sweep interval the faster that IP can be reallocated/will stop
conflicting. Network usage/CPU usage during sweeps is almost 0.

**FillARPd is fully compatible with any ARPable interfaces, this includes Ethernet, Infiniband, WiFI and others**

# Usage
fillarpd can be run in 3 main ways
- cli - Arguments passed on the command line
- systemd - Configuration enviornment variables in /etc/default/fillarpd
- docker-compose - Configuration passed as enviornment variables, copy dotenv-template to .env

### Arguments
These can be passed in via environment variables or cli flags
```
--interface/INTERFACE The interface you want to listen for ARP on  
--sourceip/SOURCE_IP The source ip for the route (in case you have multiple IPs per interface or weird network setup)  (e.g 192.168.1.1)
--network/NETWORK The subnet (e.g 192.168.1.1/24) for sweeps and constraint checking
--sweepinterval/SWEEP_INTERVAL The sweep gap in seconds, recommended to be around 60
--threads/THREADS The number of threads for sweeping.
```

# Deploying

### System Packages/systemd
**Dependencies**: libbpf & libpcap shared libraries
This project uses a Makefile to install a systemd service and build fillarpd automatically.
The Makefile has the usual options for Makefile only builds (prefix, systemddir, destdir, etc) and it is recommended to look at the top of the Makefile for more info
at the top of the make. 
```bash
git clone https://github.com/rahulc07/fillarpd.git
cd fillarpd
# Grab the release version
git checkout tags/1.0

# Install
make && make install
# edit the config file in /etc/default/fillarpd
# Reload systemd and enable the service
systemctl daemon-reload && systemctl restart fillarpd
```

### Docker
You can also deploy with docker, edit docker-compose.yml and run docker compsoe up -d

# DHCP
Most modern DHCP servers do not support IPoIB without a relay or some sort of patch.

https://github.com/rahulc07/stork-and-kea/

This Docker based DHCP + Manager has been patched for IB support. You can use this and assign the IPoIB interface a second IP on the same subnet as the Ethernet interface and use 1 pool (add both to the main listening interfaces) as long as it is on the subnet it will use the same pool as the ethernet one (even if it's not in the POOL interface listeners). 


If you can only use 1 ip assign a dummy IP to the IPoIB interface and assign it the same IP as the ethernet interface as a /32. Then put something like, this tells Kea what interfaces should bind what IP.
```json
"interfaces-config": {
     ...
    "service-sockets-require-all": true,
    "interfaces": [ "eth0/10.2.0.1", "ibs1/192.168.4.1" ],
    ...
},
```
in kea-config/kea-dhcp4.conf

Then tell Kea to force listen on ibs1
```json
"subnet4": [
        ...
        "interface": "ibs1",
        ...
]
```

Finally, send option 54 to ensure that dhcp unicast requests go through
```json
...
        "pools": [
          {
            "option-data": [
              {
                "always-send": false,
                "code": 54,
                "csv-format": true,
                "data": "10.2.0.1",
                "name": "dhcp-server-identifier",
                "never-send": false,
                "space": "dhcp4"
              }
            ],
...
// This option can be configured in stork
```
> The rationale for this is that Kea will automatically listen on interfaces in the subnet you are trying to dhcp for. The interface option causes it to force listen on the Infiniband interface that has a dummy IP. If you don't do this the Kernel won't know what 10.2.0.1 to actually bind. Again this is not needed if you use a second interface with a second IP, just listen on the ethernet interface with no dummy ip. 

In the interfaces config where 192.168.4.1 is the dummy address and 10.2.0.1 is the real address.
Running ip a show dev ibs1 should return something like
```
5: ibs1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 2044 qdisc mq state UP group default qlen 256
    link/infiniband
    altname ibp4s0
    inet 10.2.0.1/32 scope global noprefixroute ibs1
       valid_lft forever preferred_lft forever
    inet 192.168.4.1/32 scope global noprefixroute ibs1
       valid_lft forever preferred_lft forever
```
As long as you use /32s for both it should not conflict with routes. However keep in mind with fillarpd in general you lose the ability to ARP for the IB devices on your subnet, meaning you are relying on the routing table that fillarpd creates. What this means is that the bridge server won't see new IP updates until the node reaches out with it's IP because of some other ARP request or until the sweep (whichever comes first). For 99% of scenarios this is fine  

