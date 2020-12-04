#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <nvml.h>

void fail ()
{
    nvmlReturn_t result;
    result = nvmlShutdown ();
    if (NVML_SUCCESS != result)
        printf ("Error: Failed to shutdown NVML: %s\n", nvmlErrorString (result));
    exit (-1);
}

// source: https://github.com/gpeled/tensors/blob/master/gpuutil.c
int showUtilization (unsigned int i, nvmlDevice_t device, nvmlSamplingType_t type, int sleepInterval, int iterations)
{
    nvmlReturn_t result;

    unsigned long long lastSeenTimeStamp = 0;
    nvmlValueType_t sampleValType;
    unsigned int sampleCount = 1;
    nvmlSample_t samples;
    int sum = 0;

    for (int i = 0; i < iterations; i++) {
        //printf("About to query device utilization. sampleCount=%d\n",sampleCount);
        result = nvmlDeviceGetSamples (device, type,
                                       lastSeenTimeStamp, &sampleValType, &sampleCount, &samples);
        if (NVML_SUCCESS != result) {
            printf ("Error: Failed to get samples for device %i: %s\n", i, nvmlErrorString (result));
            fail ();
        }
        int util = samples.sampleValue.uiVal;
        sum += util;
        //printf("Iteration %d. GPU utilization: %d\n",i,util );
        if ((iterations - i) > 1) sleep (sleepInterval);
    }
    int average = sum / iterations;
    return average;
}

int main (int argc, char *argv[])
{
    nvmlReturn_t result;
    unsigned int device_count, i;

    // First initialize NVML library
    result = nvmlInit ();

    if (NVML_SUCCESS != result) {
        printf ("Error: Failed to initialize NVML: %s\n", nvmlErrorString (result));
        return 1;
    }
    result = nvmlDeviceGetCount (&device_count);

    if (NVML_SUCCESS != result) {
        printf ("Error Failed to query device count: %s\n", nvmlErrorString (result));
        fail();
    }

    if (argc == 1) {
        for (i = 0; i < device_count; i++) {
                nvmlDevice_t device;
                char name[64];

                result = nvmlDeviceGetHandleByIndex (i, &device);
                if (NVML_SUCCESS != result) {
                    printf ("Error: failed to get handle for device %i: %s\n", i, nvmlErrorString (result));
                    fail();
                }

                result = nvmlDeviceGetName (device, name, sizeof (name) / sizeof (name[0]));
                if (NVML_SUCCESS != result) {
                    printf ("Error: failed to get name of device %i: %s\n", i, nvmlErrorString (result));
                    fail();
                }

                nvmlSamplingType_t type = NVML_GPU_UTILIZATION_SAMPLES;
                int util = showUtilization (i, device, type, 1, 1);
                printf ("%d-%s-gpu_util-%d\n", i, name,  util);

        //        type = NVML_TOTAL_POWER_SAMPLES;
        //        util = showUtilization (i, device, type, 1, 1);
        //        printf ("%d-%s-total_power-%d\n", i, name , util);
        //
        //        type = NVML_MEMORY_UTILIZATION_SAMPLES;
        //        util = showUtilization (i, device, type, 1, 1);
        //        printf ("%d-%s-memory_util-%d\n", i, name , util);

        }
    } else if(argc == 2) {
        nvmlDevice_t device;
        char name[64];
        int i = atoi(argv[1]);

        result = nvmlDeviceGetHandleByIndex (i, &device);
        if (NVML_SUCCESS != result) {
            printf ("Error: failed to get handle for device %i: %s\n", i, nvmlErrorString (result));
            fail();
        }

        result = nvmlDeviceGetName (device, name, sizeof (name) / sizeof (name[0]));
        if (NVML_SUCCESS != result) {
            printf ("Error: failed to get name of device %i: %s\n", i, nvmlErrorString (result));
            fail();
        }

        nvmlSamplingType_t type = NVML_GPU_UTILIZATION_SAMPLES;
        int util = showUtilization (i, device, type, 1, 1);
        printf ("%d-%s-gpu_util-%d\n", i, name,  util);
    } else {
        printf("Error: only zero or one argument supported");
    }



    result = nvmlShutdown ();

    if (NVML_SUCCESS != result)
        printf ("Error: Failed to shutdown NVML: %s\n", nvmlErrorString (result));

    return 0;
}
