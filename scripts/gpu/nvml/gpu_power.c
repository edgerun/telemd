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

                int power;
                result = nvmlDeviceGetPowerUsage (device, &power);
                printf ("%d-%s-gpu_power-%d\n", i, name, power);
        }
    } else if (argc == 2) {
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

        int power;
        result = nvmlDeviceGetPowerUsage (device, &power);
        printf ("%d-%s-gpu_power-%d\n", i, name, power);
    } else {
        printf("Error: only zero or one argument supported");
    }


    result = nvmlShutdown ();

    if (NVML_SUCCESS != result)
        printf ("Error: Failed to shutdown NVML: %s\n", nvmlErrorString (result));

    return 0;
}
