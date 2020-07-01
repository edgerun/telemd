FROM nvidia/cuda as build

RUN apt update -y && apt install make 
WORKDIR /usr/local/cuda/nvml/example

RUN rm example.c
COPY gpu_util.c example.c
RUN make
RUN mv example /home/gpu_util

RUN make clean

RUN rm example.c
COPY gpu_freq.c example.c
RUN make
RUN mv example /home/gpu_freq

RUN make clean

RUN rm example.c
COPY list_gpus.c example.c
RUN make
RUN mv example /home/list_gpus

FROM debian:stretch-slim
COPY --from=build /home/gpu_freq /
COPY --from=build /home/gpu_util /
COPY --from=build /home/list_gpus /
