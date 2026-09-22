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
#pragma comment(lib, "ole32.lib")

struct CaptureDeviceInfo
{
    std::wstring name;
    IMFActivate* activate = nullptr;
};

struct CaptureFormatInfo
{
    uint32_t width = 0;
    uint32_t height = 0;

    uint32_t fpsNumerator = 0;
    uint32_t fpsDenominator = 1;

    std::wstring subtype;
};

struct ViewfinderCapture
{
    bool initialized = false;
    bool opened = false;

    std::vector<CaptureDeviceInfo> devices;

    IMFSourceReader* reader = nullptr;
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

        HRESULT nameHr = devices[i]->GetAllocatedString(
            MF_DEVSOURCE_ATTRIBUTE_FRIENDLY_NAME,
            &name,
            &nameLength
        );

        CaptureDeviceInfo info;

        if (SUCCEEDED(nameHr) && name)
        {
            info.name = name;

            CoTaskMemFree(name);
        }
        else
        {
            info.name = L"Unknown Video Device";
        }

        info.activate = devices[i];

        // Transfer ownership to CaptureDeviceInfo.
        devices[i] = nullptr;

        capture->devices.push_back(
            std::move(info)
        );
    }

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

static std::wstring GUIDToString(
    const GUID& guid
)
{
    LPOLESTR string = nullptr;

    HRESULT hr = StringFromCLSID(
        guid,
        &string
    );

    if (FAILED(hr) || !string)
    {
        return L"Unknown";
    }

    std::wstring result(string);

    CoTaskMemFree(string);

    return result;
}

static HRESULT GetFormat(
    IMFMediaType* mediaType,
    CaptureFormatInfo& format
)
{
    if (!mediaType)
    {
        return E_INVALIDARG;
    }

    UINT32 width = 0;
    UINT32 height = 0;

    HRESULT hr = MFGetAttributeSize(
        mediaType,
        MF_MT_FRAME_SIZE,
        &width,
        &height
    );

    if (FAILED(hr))
    {
        return hr;
    }

    UINT32 numerator = 0;
    UINT32 denominator = 1;

    hr = MFGetAttributeRatio(
        mediaType,
        MF_MT_FRAME_RATE,
        &numerator,
        &denominator
    );

    if (FAILED(hr))
    {
        return hr;
    }

    GUID subtype{};

    hr = mediaType->GetGUID(
        MF_MT_SUBTYPE,
        &subtype
    );

    if (FAILED(hr))
    {
        return hr;
    }

    format.width = width;
    format.height = height;

    format.fpsNumerator = numerator;
    format.fpsDenominator = denominator;

    format.subtype = GUIDToString(subtype);

    return S_OK;
}

static HRESULT EnumerateFormats(
    ViewfinderCapture* capture,
    uint32_t deviceIndex,
    std::vector<CaptureFormatInfo>& formats
)
{
    if (!capture)
    {
        return E_INVALIDARG;
    }

    if (deviceIndex >= capture->devices.size())
    {
        return MF_E_NOT_FOUND;
    }

    formats.clear();

    IMFMediaSource* source = nullptr;

    HRESULT hr =
        capture->devices[deviceIndex].activate->ActivateObject(
            __uuidof(IMFMediaSource),
            reinterpret_cast<void**>(&source)
        );

    if (FAILED(hr))
    {
        return hr;
    }

    IMFPresentationDescriptor* presentationDescriptor = nullptr;

    hr = source->CreatePresentationDescriptor(
        &presentationDescriptor
    );

    if (FAILED(hr))
    {
        source->Release();
        return hr;
    }

    DWORD streamCount = 0;

    hr = presentationDescriptor->GetStreamDescriptorCount(
        &streamCount
    );

    if (FAILED(hr))
    {
        presentationDescriptor->Release();
        source->Release();

        return hr;
    }

    for (DWORD streamIndex = 0;
         streamIndex < streamCount;
         ++streamIndex)
    {
        BOOL selected = FALSE;

        IMFStreamDescriptor* streamDescriptor = nullptr;

        hr = presentationDescriptor->GetStreamDescriptorByIndex(
            streamIndex,
            &selected,
            &streamDescriptor
        );

        if (FAILED(hr))
        {
            continue;
        }

        IMFMediaTypeHandler* handler = nullptr;

        hr = streamDescriptor->GetMediaTypeHandler(
            &handler
        );

        if (FAILED(hr))
        {
            streamDescriptor->Release();
            continue;
        }

        GUID majorType{};

        hr = handler->GetMajorType(
            &majorType
        );

        if (FAILED(hr))
        {
            handler->Release();
            streamDescriptor->Release();

            continue;
        }

        if (majorType != MFMediaType_Video)
        {
            handler->Release();
            streamDescriptor->Release();

            continue;
        }

        DWORD typeCount = 0;

        hr = handler->GetMediaTypeCount(
            &typeCount
        );

        if (SUCCEEDED(hr))
        {
            for (DWORD typeIndex = 0;
                 typeIndex < typeCount;
                 ++typeIndex)
            {
                IMFMediaType* mediaType = nullptr;

                hr = handler->GetMediaTypeByIndex(
                    typeIndex,
                    &mediaType
                );

                if (FAILED(hr))
                {
                    continue;
                }

                CaptureFormatInfo format;

                if (SUCCEEDED(
                    GetFormat(
                        mediaType,
                        format
                    )
                ))
                {
                    formats.push_back(
                        std::move(format)
                    );
                }

                mediaType->Release();
            }
        }

        handler->Release();
        streamDescriptor->Release();
    }

    presentationDescriptor->Release();
    source->Release();

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

VIEWFINDER_API int32_t ViewfinderCaptureGetFormatCount(
    ViewfinderCapture* capture,
    uint32_t deviceIndex,
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

    std::vector<CaptureFormatInfo> formats;

    HRESULT hr = EnumerateFormats(
        capture,
        deviceIndex,
        formats
    );

    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    *outCount = static_cast<uint32_t>(
        formats.size()
    );

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderCaptureGetFormat(
    ViewfinderCapture* capture,
    uint32_t deviceIndex,
    uint32_t formatIndex,
    ViewfinderCaptureFormat* outFormat
)
{
    if (!capture || !outFormat)
    {
        return E_INVALIDARG;
    }

    if (!capture->initialized)
    {
        return MF_E_NOT_INITIALIZED;
    }

    std::vector<CaptureFormatInfo> formats;

    HRESULT hr = EnumerateFormats(
        capture,
        deviceIndex,
        formats
    );

    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    if (formatIndex >= formats.size())
    {
        return MF_E_NOT_FOUND;
    }

    const auto& format =
        formats[formatIndex];

    ZeroMemory(
        outFormat,
        sizeof(ViewfinderCaptureFormat)
    );

    outFormat->index = formatIndex;

    outFormat->width = format.width;
    outFormat->height = format.height;

    outFormat->fpsNumerator =
        format.fpsNumerator;

    outFormat->fpsDenominator =
        format.fpsDenominator;

    wcsncpy_s(
        outFormat->subtype,
        format.subtype.c_str(),
        _TRUNCATE
    );

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderCaptureOpen(
    ViewfinderCapture* capture,
    uint32_t deviceIndex,
    uint32_t formatIndex
)
{
    if (!capture)
    {
        return E_INVALIDARG;
    }

    if (!capture->initialized)
    {
        return MF_E_NOT_INITIALIZED;
    }

    if (deviceIndex >= capture->devices.size())
    {
        return MF_E_NOT_FOUND;
    }

    // Close an existing stream.
    ViewfinderCaptureClose(
        capture
    );

    /*
     * Get the requested media type.
     *
     * We enumerate the native media types again because
     * IMFSourceReader works with the actual IMFMediaType
     * object, not our simplified CaptureFormatInfo.
     */

    IMFMediaSource* source = nullptr;

    HRESULT hr =
        capture->devices[deviceIndex].activate->ActivateObject(
            __uuidof(IMFMediaSource),
            reinterpret_cast<void**>(&source)
        );

    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    IMFPresentationDescriptor* presentationDescriptor = nullptr;

    hr = source->CreatePresentationDescriptor(
        &presentationDescriptor
    );

    if (FAILED(hr))
    {
        source->Release();

        return static_cast<int32_t>(hr);
    }

    DWORD streamCount = 0;

    hr = presentationDescriptor->GetStreamDescriptorCount(
        &streamCount
    );

    if (FAILED(hr))
    {
        presentationDescriptor->Release();
        source->Release();

        return static_cast<int32_t>(hr);
    }

    IMFMediaType* selectedMediaType = nullptr;

    bool found = false;

    for (DWORD streamIndex = 0;
         streamIndex < streamCount && !found;
         ++streamIndex)
    {
        BOOL selected = FALSE;

        IMFStreamDescriptor* streamDescriptor = nullptr;

        hr = presentationDescriptor->GetStreamDescriptorByIndex(
            streamIndex,
            &selected,
            &streamDescriptor
        );

        if (FAILED(hr))
        {
            continue;
        }

        IMFMediaTypeHandler* handler = nullptr;

        hr = streamDescriptor->GetMediaTypeHandler(
            &handler
        );

        if (FAILED(hr))
        {
            streamDescriptor->Release();

            continue;
        }

        GUID majorType{};

        hr = handler->GetMajorType(
            &majorType
        );

        if (FAILED(hr) ||
            majorType != MFMediaType_Video)
        {
            handler->Release();
            streamDescriptor->Release();

            continue;
        }

        DWORD typeCount = 0;

        hr = handler->GetMediaTypeCount(
            &typeCount
        );

        if (SUCCEEDED(hr))
        {
            uint32_t currentFormatIndex = 0;

            for (DWORD typeIndex = 0;
                 typeIndex < typeCount;
                 ++typeIndex)
            {
                IMFMediaType* mediaType = nullptr;

                hr = handler->GetMediaTypeByIndex(
                    typeIndex,
                    &mediaType
                );

                if (FAILED(hr))
                {
                    continue;
                }

                CaptureFormatInfo format;

                if (SUCCEEDED(
                    GetFormat(
                        mediaType,
                        format
                    )
                ))
                {
                    if (currentFormatIndex == formatIndex)
                    {
                        selectedMediaType = mediaType;
                        found = true;

                        break;
                    }

                    currentFormatIndex++;
                }

                mediaType->Release();
            }
        }

        handler->Release();
        streamDescriptor->Release();
    }

    if (!found || !selectedMediaType)
    {
        presentationDescriptor->Release();
        source->Release();

        return MF_E_NOT_FOUND;
    }

    /*
     * Create the Source Reader.
     */

    IMFAttributes* attributes = nullptr;

    hr = MFCreateAttributes(
        &attributes,
        2
    );

    if (FAILED(hr))
    {
        selectedMediaType->Release();
        presentationDescriptor->Release();
        source->Release();

        return static_cast<int32_t>(hr);
    }

    /*
     * We want video only.
     */

    hr = attributes->SetUINT32(
        MF_READWRITE_DISABLE_CONVERTERS,
        FALSE
    );

    if (FAILED(hr))
    {
        attributes->Release();
        selectedMediaType->Release();
        presentationDescriptor->Release();
        source->Release();

        return static_cast<int32_t>(hr);
    }

    hr = MFCreateSourceReaderFromMediaSource(
        source,
        attributes,
        &capture->reader
    );

    attributes->Release();

    if (FAILED(hr))
    {
        selectedMediaType->Release();
        presentationDescriptor->Release();
        source->Release();

        capture->reader = nullptr;

        return static_cast<int32_t>(hr);
    }

    /*
     * Set the requested native media type.
     */

    hr = capture->reader->SetCurrentMediaType(
        MF_SOURCE_READER_FIRST_VIDEO_STREAM,
        nullptr,
        selectedMediaType
    );

    selectedMediaType->Release();
    presentationDescriptor->Release();
    source->Release();

    if (FAILED(hr))
    {
        capture->reader->Release();
        capture->reader = nullptr;

        return static_cast<int32_t>(hr);
    }

    capture->opened = true;

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderCaptureIsOpen(
    ViewfinderCapture* capture
)
{
    if (!capture)
    {
        return E_INVALIDARG;
    }

    return capture->opened ? 1 : 0;
}

VIEWFINDER_API void ViewfinderCaptureClose(
    ViewfinderCapture* capture
)
{
    if (!capture)
    {
        return;
    }

    if (capture->reader)
    {
        capture->reader->Release();
        capture->reader = nullptr;
    }

    capture->opened = false;
}

VIEWFINDER_API int32_t ViewfinderCaptureReadFrame(
    ViewfinderCapture* capture,
    uint8_t* buffer,
    uint32_t bufferSize,
    uint32_t* outDataSize,
    uint32_t* outWidth,
    uint32_t* outHeight,
    uint32_t* outStride
)
{
    if (!capture ||
        !buffer ||
        !outDataSize ||
        !outWidth ||
        !outHeight ||
        !outStride)
    {
        return E_INVALIDARG;
    }

    if (!capture->opened || !capture->reader)
    {
        return MF_E_NOT_INITIALIZED;
    }

    *outDataSize = 0;
    *outWidth = 0;
    *outHeight = 0;
    *outStride = 0;

    DWORD streamFlags = 0;
    LONGLONG timestamp = 0;

    IMFSample* sample = nullptr;

    HRESULT hr = capture->reader->ReadSample(
        MF_SOURCE_READER_FIRST_VIDEO_STREAM,
        0,
        nullptr,
        &streamFlags,
        &timestamp,
        &sample
    );

    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    // The source may temporarily have no sample.
    if (streamFlags & MF_SOURCE_READERF_ENDOFSTREAM)
    {
        if (sample)
        {
            sample->Release();
        }

        return MF_E_END_OF_STREAM;
    }

    if (streamFlags & MF_SOURCE_READERF_CURRENTMEDIATYPECHANGED)
    {
        if (sample)
        {
            sample->Release();
        }

        return S_FALSE;
    }

    if (!sample)
    {
        return S_FALSE;
    }

    IMFMediaBuffer* mediaBuffer = nullptr;

    hr = sample->ConvertToContiguousBuffer(
        &mediaBuffer
    );

    if (FAILED(hr))
    {
        sample->Release();

        return static_cast<int32_t>(hr);
    }

    BYTE* data = nullptr;
    DWORD maxLength = 0;
    DWORD currentLength = 0;

    hr = mediaBuffer->Lock(
        &data,
        &maxLength,
        &currentLength
    );

    if (FAILED(hr))
    {
        mediaBuffer->Release();
        sample->Release();

        return static_cast<int32_t>(hr);
    }

    if (currentLength > bufferSize)
    {
        mediaBuffer->Unlock();
        mediaBuffer->Release();
        sample->Release();

        return HRESULT_FROM_WIN32(
            ERROR_INSUFFICIENT_BUFFER
        );
    }

    memcpy(
        buffer,
        data,
        currentLength
    );

    mediaBuffer->Unlock();

    mediaBuffer->Release();
    sample->Release();

    /*
     * Get the current media type so Go knows
     * the dimensions of the frame.
     */

    IMFMediaType* mediaType = nullptr;

    hr = capture->reader->GetCurrentMediaType(
        MF_SOURCE_READER_FIRST_VIDEO_STREAM,
        &mediaType
    );

    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    UINT32 width = 0;
    UINT32 height = 0;

    hr = MFGetAttributeSize(
        mediaType,
        MF_MT_FRAME_SIZE,
        &width,
        &height
    );

    if (FAILED(hr))
    {
        mediaType->Release();

        return static_cast<int32_t>(hr);
    }

    /*
     * Calculate the stride.

     * For the first implementation we assume a
     * tightly packed buffer.
     *
     * We'll handle real media-type stride correctly
     * in a later step.
     */

    GUID subtype{};

    hr = mediaType->GetGUID(
        MF_MT_SUBTYPE,
        &subtype
    );

    if (FAILED(hr))
    {
        mediaType->Release();

        return static_cast<int32_t>(hr);
    }

    LONG stride = 0;

    if (subtype == MFVideoFormat_NV12)
    {
        stride = static_cast<LONG>(width);
    }
    else
    {
        hr = MFGetStrideForBitmapInfoHeader(
            subtype.Data1,
            width,
            &stride
        );

        if (FAILED(hr))
        {
            stride = 0;
        }
    }

    mediaType->Release();

    *outDataSize = currentLength;
    *outWidth = width;
    *outHeight = height;
    *outStride = static_cast<uint32_t>(
        stride > 0 ? stride : 0
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

    ViewfinderCaptureClose(
        capture
    );

    ReleaseDevices(
        capture
    );

    if (capture->initialized)
    {
        MFShutdown();

        capture->initialized = false;
    }

    delete capture;
}
