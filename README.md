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

Parameters
----------

The `telemc` command allows the following parameters via environment variables.

| Variable | Default | Description |
|---|---|---|
| `telemc_node_name`    | `$HOST`       | The node name determines the value for `<hostname>` in the topics |
| `telemc_redis_host`   | `localhost`   | The redis host to connect to |
| `telemc_redis_port`   | `6379`        | The redis port to connect to |
| `telemc_redis_url`    |               | Can be used to specify the redis URL (e.g., `redis://localhost:1234`). Overwrites anything set to `telemc_redis_host`.
| `telemc_net_devices`  | all           | A list of network devices to be monitored, e.g. `wlan0 eth0`. Monitors all devices per default |
| `telemc_disk_devices` | all           | A list of block devices to be monitored, e.g. `sda sdc sdd0`. Monitors all devices per default |
