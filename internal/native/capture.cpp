#define VIEWFINDER_NATIVE_EXPORTS

#include "capture.h"

#include <windows.h>
#include <mfapi.h>
#include <mfidl.h>
#include <mfreadwrite.h>

#include <new>

#pragma comment(lib, "mfplat.lib")
#pragma comment(lib, "mf.lib")
#pragma comment(lib, "mfuuid.lib")
#pragma comment(lib, "mfreadwrite.lib")

struct ViewfinderCapture
{
    bool initialized = false;
};

VIEWFINDER_API int32_t ViewfinderCaptureCreate(
    ViewfinderCapture** outCapture
)
{
    if (!outCapture)
    {
        return E_INVALIDARG;
    }

    *outCapture = nullptr;

    ViewfinderCapture* capture =
        new (std::nothrow) ViewfinderCapture();

    if (!capture)
    {
        return E_OUTOFMEMORY;
    }

    *outCapture = capture;

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderCaptureInitialize(
    ViewfinderCapture* capture
)
{
    if (!capture)
    {
        return E_INVALIDARG;
    }

    if (capture->initialized)
    {
        return S_OK;
    }

    HRESULT hr = MFStartup(
        MF_VERSION,
        MFSTARTUP_FULL
    );

    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    capture->initialized = true;

    return S_OK;
}

VIEWFINDER_API void ViewfinderCaptureDestroy(
    ViewfinderCapture* capture
)
{
    if (!capture)
    {
        return;
    }

    if (capture->initialized)
    {
        MFShutdown();
        capture->initialized = false;
    }

    delete capture;
}