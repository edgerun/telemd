FROM nvidia/cuda as build

RUN apt update -y && apt install make 
WORKDIR /usr/local/cuda/nvml/example

RUN rm example.c
COPY gpu_measurements.c example.c
RUN make
RUN mv example /home/gpu_measurements

RUN make clean

RUN rm example.c
COPY list_gpus.c example.c
RUN make
RUN mv example /home/list_gpus

FROM debian:stretch-slim
COPY --from=build /home/gpu_measurements /
COPY --from=build /home/list_gpus /
