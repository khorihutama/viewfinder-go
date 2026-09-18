#include "capture.h"

#include <windows.h>

#include <mfapi.h>
#include <mfidl.h>
#include <mfreadwrite.h>
#include <mferror.h>

#include <new>
#include <string>
#include <vector>

#pragma comment(lib, "mfplat.lib")
#pragma comment(lib, "mf.lib")
#pragma comment(lib, "mfuuid.lib")
#pragma comment(lib, "mfreadwrite.lib")

struct CaptureDeviceInfo
{
    std::wstring name;
    IMFActivate* activate = nullptr;
};

struct ViewfinderCapture
{
    bool initialized = false;

    std::vector<CaptureDeviceInfo> devices;
};

static void ReleaseDevices(
    ViewfinderCapture* capture
)
{
    if (!capture)
    {
        return;
    }

    for (auto& device : capture->devices)
    {
        if (device.activate)
        {
            device.activate->Release();
            device.activate = nullptr;
        }
    }

    capture->devices.clear();
}

static HRESULT EnumerateDevices(
    ViewfinderCapture* capture
)
{
    if (!capture)
    {
        return E_INVALIDARG;
    }

    ReleaseDevices(capture);

    IMFAttributes* attributes = nullptr;

    HRESULT hr = MFCreateAttributes(
        &attributes,
        1
    );

    if (FAILED(hr))
    {
        return hr;
    }

    hr = attributes->SetGUID(
        MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE,
        MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE_VIDCAP_GUID
    );

    if (FAILED(hr))
    {
        attributes->Release();
        return hr;
    }

    IMFActivate** devices = nullptr;
    UINT32 deviceCount = 0;

    hr = MFEnumDeviceSources(
        attributes,
        &devices,
        &deviceCount
    );

    attributes->Release();

    if (FAILED(hr))
    {
        return hr;
    }

    for (UINT32 i = 0; i < deviceCount; ++i)
    {
        WCHAR* name = nullptr;
        UINT32 nameLength = 0;

        hr = devices[i]->GetAllocatedString(
            MF_DEVSOURCE_ATTRIBUTE_FRIENDLY_NAME,
            &name,
            &nameLength
        );

        std::wstring deviceName;

        if (SUCCEEDED(hr) && name)
        {
            deviceName = name;
            CoTaskMemFree(name);
        }
        else
        {
            deviceName = L"Unknown Video Device";
        }

        CaptureDeviceInfo info;

        info.name = deviceName;
        info.activate = devices[i];

        // We now own this reference.
        devices[i] = nullptr;

        capture->devices.push_back(
            std::move(info)
        );
    }

    // Release anything not transferred.
    for (UINT32 i = 0; i < deviceCount; ++i)
    {
        if (devices[i])
        {
            devices[i]->Release();
        }
    }

    CoTaskMemFree(devices);

    return S_OK;
}

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

    hr = EnumerateDevices(
        capture
    );

    if (FAILED(hr))
    {
        MFShutdown();

        capture->initialized = false;

        return static_cast<int32_t>(hr);
    }

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderCaptureGetDeviceCount(
    ViewfinderCapture* capture,
    uint32_t* outCount
)
{
    if (!capture || !outCount)
    {
        return E_INVALIDARG;
    }

    if (!capture->initialized)
    {
        return MF_E_NOT_INITIALIZED;
    }

    *outCount = static_cast<uint32_t>(
        capture->devices.size()
    );

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderCaptureGetDevice(
    ViewfinderCapture* capture,
    uint32_t index,
    ViewfinderCaptureDevice* outDevice
)
{
    if (!capture || !outDevice)
    {
        return E_INVALIDARG;
    }

    if (!capture->initialized)
    {
        return MF_E_NOT_INITIALIZED;
    }

    if (index >= capture->devices.size())
    {
        return MF_E_NOT_FOUND;
    }

    const auto& device =
        capture->devices[index];

    ZeroMemory(
        outDevice,
        sizeof(ViewfinderCaptureDevice)
    );

    outDevice->index = index;

    wcsncpy_s(
        outDevice->name,
        device.name.c_str(),
        _TRUNCATE
    );

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

    ReleaseDevices(capture);

    if (capture->initialized)
    {
        MFShutdown();

        capture->initialized = false;
    }

    delete capture;
}