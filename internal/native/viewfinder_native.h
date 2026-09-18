#pragma once

#include <windows.h>
#include <stdint.h>

#ifdef VIEWFINDER_NATIVE_EXPORTS
#define VIEWFINDER_API __declspec(dllexport)
#else
#define VIEWFINDER_API __declspec(dllimport)
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef struct ViewfinderRenderer ViewfinderRenderer;

VIEWFINDER_API int32_t ViewfinderCreateRenderer(
    HWND hwnd,
    uint32_t width,
    uint32_t height,
    ViewfinderRenderer** outRenderer
);

VIEWFINDER_API int32_t ViewfinderRender(
    ViewfinderRenderer* renderer
);

VIEWFINDER_API void ViewfinderDestroyRenderer(
    ViewfinderRenderer* renderer
);

#ifdef __cplusplus
}
#endif