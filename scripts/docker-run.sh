docker run --privileged=true \
  --network host \
  -v /sys:/sys:ro \
  -v /proc:/proc_host \
  -e telemd_proc_mount=/proc_host \
  -e telemd_instruments_disable="kubernetes_cgrp_cpu kubernetes_cgrp_blkio kubernetes_cgrp_memory kubernetes_cgrp_net" \
  --gpus all \
  edgerun/telemd:0.9.5