docker run --privileged=true \
  --network host \
  -v /sys:/sys:ro \
  -e telemd_instruments_disable=docker_cgrp_net
  edgerun/telemd