# gnss-relay

An RTCM3 streaming relay server

The aim is to provide a low-latency mechanism for distributing an RTCM3 feed from
a remote GNSS receiver to various clients without excessive connections to the
remote device.

The server will only relay packets that have passed any initial RTCM3 header and CRC checks.

It is possible to daisy chain these services.
