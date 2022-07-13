# arbok

Arbok helps you create ephemeral HTTP tunnels and share local applications running on your machine with the outside world. It uses userspace implementation of Wireguard to setup networking between the client and server. SSL and routing is offloaded to an embedded Caddy instance running in the server.