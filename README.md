telemd
======

![Docker Image Version (latest semver)](https://img.shields.io/docker/v/edgerun/telemd-gpu?color=blue&label=Docker%20version&sort=semver)

A daemon that reports fine-grained systems runtime data into Redis

Build
-----

To build the module's binaries using your local go installation run:

    make

To build Docker images for local usage (without a go installation) run:

    make docker

*Hint*: In case there are build errors with the ARM images you might need 
to run the following command first, as described [here](https://github.com/multiarch/qemu-user-static)

`docker run --rm --privileged multiarch/qemu-user-static --reset -p yes`


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
* `ram` RAM currently used in kilobytes
* `disk` Disk I/O rate averaged in bytes/second
* `net` Network I/O rate averaged in kilobytes/second
* `load` the system load average of the last 1 and 5 minutes
* `procs` the number of processes running at the current time
* `tx_bitrate` the tx bitrate reported by `iw`. Only available for wireless interfaces
* `rx_bitrate` the rx bitrate reported by `iw`. Only available for wireless interfaces
* `signal` the signal strength reported by `iw`. Only available for wireless interfaces
* `psi_cpu` host's CPU [pressure](https://www.kernel.org/doc/html/latest/accounting/psi.html#psi)
* `psi_io` host's I/O [pressure](https://www.kernel.org/doc/html/latest/accounting/psi.html#psi)
* `psi_memory` host's memory [pressure](https://www.kernel.org/doc/html/latest/accounting/psi.html#psi)
* `docker_cgrp_cpu` the cpu usage time of individual docker containers
* `docker_cgrp_blkio` the total block io usage in bytes for individual docker containers
* `docker_cgrp_net` the total network io usage in bytes for individual docker containers as well as for each interface and `rx` and `tx`
  * I.e.: `docker_cgrp_net/<container-id>`, `docker_cgrp_net/<container-id>/<interface>`, `docker_cgrp_net/<container-id>/<interface>/[rx|tx]`
* `docker_cgrp_memory` the total memory (RAM) usage in bytes for individual docker containers
* `kubernetes_cgrp_cpu` the cpu usage time of individual Kubernetes Pod containers
* `kubernetes_cgrp_blkio` the total block io usage in bytes for individual Kubernetes Pod containers
* `kubernetes_cgrp_memory` the total memory (RAM) usage in bytes for individual Kubernetes Pod containers
* `kubernetes_cgrp_net` the total network io usage in bytes for individual Kubernetes Pod containers as well as for each interface and `rx` and `tx`
  * I.e.: `kubernetes_cgrp_net/<container-id>`, `kubernetes_cgrp_net/<container-id>/<interface>`, `kubernetes_cgrp_net/<container-id>/<interface>/[rx|tx]`
* `gpu_freq` the current clock frequency of GPUs
  * On normal Nvidia GPUs (i.e., on `amd64` systems), we use [nvmlDeviceGetClock](https://docs.nvidia.com/deploy/nvml-api/group__nvmlDeviceQueries.html#group__nvmlDeviceQueries_1g2efc4dd4096173f01d80b2a8bbfd97ad) 
  * On Jetson Boards, we read from `/sys/devices/gpu.0/devfreq/%s/cur_freq`
* `gpu_util` GPU utilization in `%`
  * On normal Nvidia GPUs (i.e., on `amd64` systems) we use [nvmlDeviceGetUtilizationRates](https://docs.nvidia.com/deploy/nvml-api/group__nvmlDeviceQueries.html#group__nvmlDeviceQueries_1g540824faa6cef45500e0d1dc2f50b321)
  * On Jetson Boards we read from `/sys/devices/gpu.0/load`
* `gpu_util_memory` GPU memory utilization of the last second in `%`
  * This is only available on normal Nvidia GPUs, see [nvmlDeviceGetUtilizationRates](https://docs.nvidia.com/deploy/nvml-api/group__nvmlDeviceQueries.html#group__nvmlDeviceQueries_1g540824faa6cef45500e0d1dc2f50b321)
  
### GPU support

Telemd enables GPU monitoring through the usage of two images `edgerun/telemd` (CPU) & `edgerun/telemd-gpu` (GPU).
Currently, the GPU version supports:

* Nvidia on `amd64`/`x86_64` (tested with CUDA 10.1)
* Nvidia Jetson Boards (tested with NX, TX2 and Nano)

Further, on `amd64` you have to run the container with:

    docker run --gpus all

to pass the GPU through into the container.

#### Attention
The `amd64` version will currently (probably) only work with CUDA 10.1 installations.
See [amd64 Dockerfile](build/package/telemd-gpu/Dockerfile.amd64)

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
| `gpu`      | [str]  | The GPUs available for monitoring (`id-name`) |
| `netspeed` | str    | LAN/WLAN speed in Mbps |

### Talking back to hosts

Telemd hosts listen on the topic

    telemcmd/<nodename>

for commands. Currently, telemd supports the following commands:

* `pause` pauses reporting of metrics
* `unpause` unpauses report of metrics
* `info` update the info keys

### Telemetry Daemon Parameters

#### Environment variables

The `telemd` command allows the following parameters via environment variables.

| Variable | Default | Description |
|---|---|---|
| `telemd_nodename`     | `$HOST`       | The node name determines the value for `<nodename>` in the topics |
| `telemd_redis_host`   | `localhost`   | The redis host to connect to |
| `telemd_redis_port`   | `6379`        | The redis port to connect to |
| `telemd_redis_url`    |               | Can be used to specify the redis URL (e.g., `redis://localhost:1234`). Overwrites anything set to `telemd_redis_host`.
| `telemd_net_devices`  | all           | A list of network devices to be monitored, e.g. `wlan0 eth0`. Monitors all devices per default |
| `telemd_disk_devices` | all           | A list of block devices to be monitored, e.g. `sda sdc sdd0`. Monitors all devices per default |
| `telemd_period_<instrument>` |        | A duration string (`1s`, `500ms`, ...) that indicates how often the given `instrument` should be probed |
| `telemd_instruments_enable`  | all    | A space seperated list of instruments to use (e.g. `"cpu freq"`), these will be the only instruments that are run (mutex with disable) |
| `telemd_instruments_disable` | none   | A space seperated list of instruments to disable, all instruments will run except for these (mutex with enable, preferred if both are set) |
| `telemd_gpu_devices`  | all           | A list of GPUs to be monitored, e.g. `0 1`. Monitors all devices per default |
| `telemd_proc_mount`    | `/proc`      | Tells telemd where the `/proc` folder is mounted into the container. |
#### Configuration

To allow the global configuration of a fleet of `telemd` instances, telemd can also be configured via ini files.
A global configuration could look like this, where each section refers to a specific `telemd_nodename`.
The `telemd` instance will look up its config in the section corresponding to its hostname.
Values outside a section will be applied first.
Environment variables will overwrite ini values.
By default, we look up the config in `/etc/telemd/config.ini`

```ini
telemd_redis_host=192.168.0.10

[myhost1]
telemd_net_devices=eth0 wlan0
telemd_disk_devices=sda

[myhost2]
telemd_net_devices=enp5s0
telemd_disk_devices=sdb
# ...
```

Run as docker container
-----------------------
Execute, or run (`./scripts/docker-run.sh`):

    docker run --privileged=true \
    --network host \
    -v /sys:/sys:ro \
    -v /proc:/proc_host \
    -e telemd_instruments_disable="kubernetes_cgrp_cpu kubernetes_cgrp_blkio kubernetes_cgrp_memory kubernetes_cgrp_net" \
    -e telemd_proc_mount=/proc_host
    edgerun/telemd
