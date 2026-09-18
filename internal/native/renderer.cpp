#include "viewfinder_native.h"

#include <windows.h>
#include <d3d11.h>
#include <dxgi.h>

#pragma comment(lib, "d3d11.lib")
#pragma comment(lib, "dxgi.lib")

static ID3D11Device* g_device = nullptr;
static ID3D11DeviceContext* g_context = nullptr;
static IDXGISwapChain* g_swapChain = nullptr;
static ID3D11RenderTargetView* g_renderTarget = nullptr;

static void ReleaseRenderer()
{
    if (g_renderTarget)
    {
        g_renderTarget->Release();
        g_renderTarget = nullptr;
    }

    if (g_swapChain)
    {
        g_swapChain->Release();
        g_swapChain = nullptr;
    }

    if (g_context)
    {
        g_context->Release();
        g_context = nullptr;
    }

    if (g_device)
    {
        g_device->Release();
        g_device = nullptr;
    }
}

VIEWFINDER_API int32_t ViewfinderRendererInitialize(
    void* hwnd,
    uint32_t width,
    uint32_t height
)
{
    if (!hwnd)
    {
        return E_INVALIDARG;
    }

    ReleaseRenderer();

    DXGI_SWAP_CHAIN_DESC swapChainDesc{};

    swapChainDesc.BufferCount = 2;

    swapChainDesc.BufferDesc.Width = width;
    swapChainDesc.BufferDesc.Height = height;

    swapChainDesc.BufferDesc.Format =
        DXGI_FORMAT_R8G8B8A8_UNORM;

    swapChainDesc.BufferUsage =
        DXGI_USAGE_RENDER_TARGET_OUTPUT;

    swapChainDesc.OutputWindow =
        static_cast<HWND>(hwnd);

    swapChainDesc.SampleDesc.Count = 1;

    swapChainDesc.Windowed = TRUE;

    D3D_FEATURE_LEVEL featureLevels[] =
    {
        D3D_FEATURE_LEVEL_11_1,
        D3D_FEATURE_LEVEL_11_0
    };

    D3D_FEATURE_LEVEL featureLevel{};

    HRESULT hr = D3D11CreateDeviceAndSwapChain(
        nullptr,
        D3D_DRIVER_TYPE_HARDWARE,
        nullptr,
        0,
        featureLevels,
        2,
        D3D11_SDK_VERSION,
        &swapChainDesc,
        &g_swapChain,
        &g_device,
        &featureLevel,
        &g_context
    );

    if (FAILED(hr))
    {
        ReleaseRenderer();

        return static_cast<int32_t>(hr);
    }

    ID3D11Texture2D* backBuffer = nullptr;

    hr = g_swapChain->GetBuffer(
        0,
        __uuidof(ID3D11Texture2D),
        reinterpret_cast<void**>(&backBuffer)
    );

    if (FAILED(hr))
    {
        ReleaseRenderer();

        return static_cast<int32_t>(hr);
    }

    hr = g_device->CreateRenderTargetView(
        backBuffer,
        nullptr,
        &g_renderTarget
    );

    backBuffer->Release();

    if (FAILED(hr))
    {
        ReleaseRenderer();

        return static_cast<int32_t>(hr);
    }

    g_context->OMSetRenderTargets(
        1,
        &g_renderTarget,
        nullptr
    );

    D3D11_VIEWPORT viewport{};

    viewport.TopLeftX = 0.0f;
    viewport.TopLeftY = 0.0f;

    viewport.Width =
        static_cast<float>(width);

    viewport.Height =
        static_cast<float>(height);

    viewport.MinDepth = 0.0f;
    viewport.MaxDepth = 1.0f;

    g_context->RSSetViewports(
        1,
        &viewport
    );

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderRendererClear(
    float red,
    float green,
    float blue,
    float alpha
)
{
    if (!g_context || !g_renderTarget)
    {
        return E_FAIL;
    }

    float color[4] =
    {
        red,
        green,
        blue,
        alpha
    };

    g_context->ClearRenderTargetView(
        g_renderTarget,
        color
    );

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderRendererPresent()
{
    if (!g_swapChain)
    {
        return E_FAIL;
    }

    HRESULT hr = g_swapChain->Present(
        1,
        0
    );

    return static_cast<int32_t>(hr);
}

VIEWFINDER_API void ViewfinderRendererDestroy()
{
    ReleaseRenderer();
}