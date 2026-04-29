# FillARPd Fill in /32s to the Route Table from Arp Snooping and Sweeping

This tool/daemon is intended to be used for Proxy Arp Setups (on the proxy arp gateway) 
where you have to see all the devices behind a specific interface even when your proxy arp host
might not necessarily be on that subnet (e.g your proxy arp host is /32 sharing an IP)
This tool allows you to scan for wthose devices and as long as they can make ARP requests back to your source ip it will fill in the route table with /32s with the devices it discovers.
It also keeps the arp tables up to date when devices are no longer up, allowing the IPs to be reallocated without conflict.

fillarpd discovers devices in 2 ways, ARP Snooping and sweeping. 
- ARP Snooping is reactive and will update within 5 milliseconds of a device making an ARP request
allowing the node on the other side of Proxy Arp to send it's data back without
noticing the route table changes.
- Sweeping is proactive, every sweep interval and at startup fillarpd will lookup all the ips in the subnet to ensure that IPs that haven't checked in are active. If they are not active their route
will be removed. The shorter the Sweep interval the faster that IP can be reallocated/will stop
conflicting. Network usage/CPU usage during sweeps is almost 0.