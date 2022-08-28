# broadcasthost

Dirty little helper application for detecting if I'm on a local or remote network.

This application runs as both a client and server. The server component listens on
a central machine inside my home network, if it reads a message on UDP broadcast, it
replies with the hostname.

The client can then use this to determine if it is on a local network (service running and available)
or a remote network (service unavailable) and reconfigure apps/services appropriately.
