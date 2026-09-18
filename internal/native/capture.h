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

typedef struct ViewfinderCaptureFormat
{
    uint32_t index;

    uint32_t width;
    uint32_t height;

    uint32_t fpsNumerator;
    uint32_t fpsDenominator;

    wchar_t subtype[64];

} ViewfinderCaptureFormat;


/* Lifecycle */

VIEWFINDER_API int32_t ViewfinderCaptureCreate(
    ViewfinderCapture** outCapture
);

VIEWFINDER_API int32_t ViewfinderCaptureInitialize(
    ViewfinderCapture* capture
);

VIEWFINDER_API void ViewfinderCaptureDestroy(
    ViewfinderCapture* capture
);


/* Devices */

VIEWFINDER_API int32_t ViewfinderCaptureGetDeviceCount(
    ViewfinderCapture* capture,
    uint32_t* outCount
);

VIEWFINDER_API int32_t ViewfinderCaptureGetDevice(
    ViewfinderCapture* capture,
    uint32_t index,
    ViewfinderCaptureDevice* outDevice
);


/* Formats */

VIEWFINDER_API int32_t ViewfinderCaptureGetFormatCount(
    ViewfinderCapture* capture,
    uint32_t deviceIndex,
    uint32_t* outCount
);

VIEWFINDER_API int32_t ViewfinderCaptureGetFormat(
    ViewfinderCapture* capture,
    uint32_t deviceIndex,
    uint32_t formatIndex,
    ViewfinderCaptureFormat* outFormat
);


/* Capture stream */

VIEWFINDER_API int32_t ViewfinderCaptureOpen(
    ViewfinderCapture* capture,
    uint32_t deviceIndex,
    uint32_t formatIndex
);

VIEWFINDER_API int32_t ViewfinderCaptureIsOpen(
    ViewfinderCapture* capture
);

VIEWFINDER_API void ViewfinderCaptureClose(
    ViewfinderCapture* capture
);


#ifdef __cplusplus
}
#endif