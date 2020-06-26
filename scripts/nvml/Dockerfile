FROM nvidia/cuda as build

RUN apt update -y && apt install make 
WORKDIR /usr/local/cuda/nvml/example

RUN rm example.c
COPY gpu_freq.c .
COPY gpu_util.c .

# make gpu_freq.c
RUN mv gpu_freq.c example.c
RUN make
RUN mv example /home/gpu_freq
RUN rm example.c
RUN make clean

# make get_util.c
RUN mv gpu_util.c example.c
RUN make
RUN mv example /home/gpu_util

# FROM ubuntu
#FROM frolvlad/alpine-glibc
FROM debian:stretch-slim
COPY --from=build /home/gpu_freq /
COPY --from=build /home/gpu_util /
