go-telemd
=========

A daemon that reports fine-grained systems runtime data into Redis

Build
-----

To build the module's binaries using your local go installation run:

    make

To build Docker images for local usage (without a go installation) run:

    make docker

Usage
-----

### Topic schema

telemd reports telemetry data into (Redis) topics.

The topic schema is as follows. It starts with the keyword `telem`, followed by the node name that reports the
telemetry, the specific metric being reported, and optionally the subsystem (e.g., a specific network device or disk).

    telem/<nodename>/<metric>[/<subsystem>]

For example, the CPU utilization of CPU core 0 host `rpi0` could be reported as:

    telem/rpi0/cpu/0

Or it may report an aggregate value into

    telem/rpi0/cpu

#### Instruments

The default telemd runs the following instruments:

* `cpu` The CPU utilization of the last 0.5 seconds in `%`
* `freq` The sum of clock frequencies of the main CPUs
* `disk` Disk I/O rate averaged in bytes/second
* `net` Network I/O rate averaged in bytes/second
* `load` the system load average of the last 1 and 5 minutes

### Info keys

When a telemetry daemon starts, it writes static information about its host into the Redis key 
`telemd.info:<nodename>`.
It is a Redis hash has the following keys:

| Key | Type | Description |
|---|---|---|
| `arch`     | str    | the processor architecture (`arm32`, `amd64`, ...) |
| `cpus`     | int    | number of processors |
| `ram`      | int    | maximal available RAM in kilobytes |
| `boot`     | int    | UNIX timestamp of when the node was last booted |
| `disk`     | [str]  | The disk devices available for monitoring |
| `net`      | [str]  | The network devices available for monitoring |
| `hostname` | str    | The real hostname |

### Talking back to hosts

Telemd hosts listen on the topic

    telemcmd/<nodename>

for commands. Currently, telemd supports the following commands:

* `pause` pauses reporting of metrics
* `unpause` unpauses report of metrics
* `info` update the info keys

### Telemetry Daemon Parameters

The `telemd` command allows the following parameters via environment variables.

| Variable | Default | Description |
|---|---|---|
| `telemd_nodename`     | `$HOST`       | The node name determines the value for `<nodename>` in the topics |
| `telemd_redis_host`   | `localhost`   | The redis host to connect to |
| `telemd_redis_port`   | `6379`        | The redis port to connect to |
| `telemd_redis_url`    |               | Can be used to specify the redis URL (e.g., `redis://localhost:1234`). Overwrites anything set to `telemd_redis_host`.
| `telemd_net_devices`  | all           | A list of network devices to be monitored, e.g. `wlan0 eth0`. Monitors all devices per default |
| `telemd_disk_devices` | all           | A list of block devices to be monitored, e.g. `sda sdc sdd0`. Monitors all devices per default |
