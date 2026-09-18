#pragma once

#include <stdint.h>

#ifdef VIEWFINDER_NATIVE_EXPORTS
#define VIEWFINDER_API __declspec(dllexport)
#else
#define VIEWFINDER_API __declspec(dllimport)
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef struct ViewfinderCapture ViewfinderCapture;

typedef struct ViewfinderCaptureDevice
{
    uint32_t index;

    wchar_t name[256];

} ViewfinderCaptureDevice;

VIEWFINDER_API int32_t ViewfinderCaptureCreate(
    ViewfinderCapture** outCapture
);

VIEWFINDER_API int32_t ViewfinderCaptureInitialize(
    ViewfinderCapture* capture
);

VIEWFINDER_API int32_t ViewfinderCaptureGetDeviceCount(
    ViewfinderCapture* capture,
    uint32_t* outCount
);

VIEWFINDER_API int32_t ViewfinderCaptureGetDevice(
    ViewfinderCapture* capture,
    uint32_t index,
    ViewfinderCaptureDevice* outDevice
);

VIEWFINDER_API void ViewfinderCaptureDestroy(
    ViewfinderCapture* capture
);

#ifdef __cplusplus
}
#endif