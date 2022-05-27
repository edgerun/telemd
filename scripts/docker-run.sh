docker run --privileged=true \
  --network host \
  -v /sys:/sys:ro \
  -v /proc:/proc \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e telemd_instruments_disable="kubernetes_cgrp_cpu kubernetes_cgrp_blkio kubernetes_cgrp_memory kubernetes_cgrp_net" \
  edgerun/telemd