#pragma once

#ifdef VIEWFINDER_NATIVE_EXPORTS
#define VIEWFINDER_API __declspec(dllexport)
#else
#define VIEWFINDER_API __declspec(dllimport)
#endif

#include <windows.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

VIEWFINDER_API int32_t ViewfinderCreateSwapChain(
    HWND hwnd,
    uint32_t width,
    uint32_t height,
    void** outSwapChain
);

VIEWFINDER_API void ViewfinderReleaseObject(
    void* object
);

#ifdef __cplusplus
}
#endif