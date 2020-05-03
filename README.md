go-telemc
=========

A symmetry telemetry push client written in go

Topic schema
------------

telemc reports telemetry data into (Redis) topics.

The topic schema is as follows. It starts with the keyword `telem`, followed by the hostname that reports the telemetry,
the specific metric being reported, and optionally the subsystem (e.g., a specific network device or disk).

    telem/<hostname>/<metric>[/<subsystem>]

For example, the CPU utilization of CPU core 0 host `rpi0` could be reported as:

    telem/rpi0/cpu/0

Or it may report an aggregate value into

    telem/rpi0/cpu

Talking back to clients
-----------------------

Clients listen on the topic

    telemcmd/<hostname>

for commands. Currently, telemc supports the following commands:

* `pause` pauses reporting of metrics
* `unpause` unpauses report of metrics
