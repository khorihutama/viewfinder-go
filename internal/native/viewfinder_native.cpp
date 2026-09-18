#define VIEWFINDER_NATIVE_EXPORTS

#include "viewfinder_native.h"

#include <d3d11.h>
#include <dxgi1_2.h>

#pragma comment(lib, "d3d11.lib")
#pragma comment(lib, "dxgi.lib")

static HRESULT CreateD3D11Device(
    ID3D11Device** outDevice,
    ID3D11DeviceContext** outContext
)
{
    if (!outDevice || !outContext)
        return E_INVALIDARG;

    *outDevice = nullptr;
    *outContext = nullptr;

    D3D_FEATURE_LEVEL featureLevels[] = {
        D3D_FEATURE_LEVEL_11_1,
        D3D_FEATURE_LEVEL_11_0
    };

    D3D_FEATURE_LEVEL selectedFeatureLevel{};

    UINT flags = D3D11_CREATE_DEVICE_BGRA_SUPPORT;

#ifdef _DEBUG
    flags |= D3D11_CREATE_DEVICE_DEBUG;
#endif

    HRESULT hr = D3D11CreateDevice(
        nullptr,
        D3D_DRIVER_TYPE_HARDWARE,
        nullptr,
        flags,
        featureLevels,
        ARRAYSIZE(featureLevels),
        D3D11_SDK_VERSION,
        outDevice,
        &selectedFeatureLevel,
        outContext
    );

    return hr;
}

VIEWFINDER_API int32_t ViewfinderCreateSwapChain(
    HWND hwnd,
    uint32_t width,
    uint32_t height,
    void** outSwapChain
)
{
    if (!hwnd || !outSwapChain)
        return E_INVALIDARG;

    *outSwapChain = nullptr;

    ID3D11Device* device = nullptr;
    ID3D11DeviceContext* context = nullptr;

    HRESULT hr = CreateD3D11Device(
        &device,
        &context
    );

    if (FAILED(hr))
        return static_cast<int32_t>(hr);

    IDXGIDevice* dxgiDevice = nullptr;

    hr = device->QueryInterface(
        __uuidof(IDXGIDevice),
        reinterpret_cast<void**>(&dxgiDevice)
    );

    if (FAILED(hr))
    {
        context->Release();
        device->Release();
        return static_cast<int32_t>(hr);
    }

    IDXGIAdapter* adapter = nullptr;

    hr = dxgiDevice->GetAdapter(&adapter);

    if (FAILED(hr))
    {
        dxgiDevice->Release();
        context->Release();
        device->Release();
        return static_cast<int32_t>(hr);
    }

    IDXGIFactory2* factory = nullptr;

    hr = adapter->GetParent(
        __uuidof(IDXGIFactory2),
        reinterpret_cast<void**>(&factory)
    );

    if (FAILED(hr))
    {
        adapter->Release();
        dxgiDevice->Release();
        context->Release();
        device->Release();
        return static_cast<int32_t>(hr);
    }

    DXGI_SWAP_CHAIN_DESC1 desc{};

    desc.Width = width;
    desc.Height = height;

    desc.Format = DXGI_FORMAT_R8G8B8A8_UNORM;

    desc.Stereo = FALSE;

    desc.SampleDesc.Count = 1;
    desc.SampleDesc.Quality = 0;

    desc.BufferUsage =
        DXGI_USAGE_RENDER_TARGET_OUTPUT;

    desc.BufferCount = 2;

    desc.Scaling =
        DXGI_SCALING_STRETCH;

    desc.SwapEffect =
        DXGI_SWAP_EFFECT_FLIP_DISCARD;

    desc.AlphaMode =
        DXGI_ALPHA_MODE_IGNORE;

    desc.Flags = 0;

    IDXGISwapChain1* swapChain = nullptr;

    hr = factory->CreateSwapChainForHwnd(
        device,
        hwnd,
        &desc,
        nullptr,
        nullptr,
        &swapChain
    );

    factory->Release();
    adapter->Release();
    dxgiDevice->Release();
    context->Release();
    device->Release();

    if (FAILED(hr))
        return static_cast<int32_t>(hr);

    *outSwapChain = swapChain;

    return S_OK;
}

VIEWFINDER_API void ViewfinderReleaseObject(
    void* object
)
{
    if (!object)
        return;

    IUnknown* unknown =
        static_cast<IUnknown*>(object);

    unknown->Release();
}