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

VIEWFINDER_API int32_t ViewfinderCaptureCreate(
    ViewfinderCapture** outCapture
);

VIEWFINDER_API int32_t ViewfinderCaptureInitialize(
    ViewfinderCapture* capture
);

VIEWFINDER_API void ViewfinderCaptureDestroy(
    ViewfinderCapture* capture
);

#ifdef __cplusplus
}
#endif