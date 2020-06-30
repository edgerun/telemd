FROM nvidia/cuda as build

RUN apt update -y && apt install make 
WORKDIR /usr/local/cuda/nvml/example

RUN rm example.c
COPY gpu_measurements.c example.c

RUN make
RUN mv example /home/gpu_measurements

FROM debian:stretch-slim
COPY --from=build /home/gpu_measurements /
