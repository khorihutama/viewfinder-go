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

VIEWFINDER_API int32_t ViewfinderRendererInitialize(
    void* hwnd,
    uint32_t width,
    uint32_t height
);

VIEWFINDER_API int32_t ViewfinderRendererPresent();

VIEWFINDER_API int32_t ViewfinderRendererResize(
    uint32_t width,
    uint32_t height
);

VIEWFINDER_API int32_t ViewfinderRendererSetFillMode(int32_t fill);

VIEWFINDER_API void ViewfinderRendererDestroy();

VIEWFINDER_API int32_t ViewfinderRendererDraw();

VIEWFINDER_API int32_t ViewfinderRendererUploadNV12(
    const uint8_t* data,
    uint32_t dataSize,
    uint32_t width,
    uint32_t height,
    uint32_t stride
);

#ifdef __cplusplus
}
#endif
