#include "viewfinder_native.h"

#include <d3d11.h>
#include <dxgi1_2.h>

#include <new>

#pragma comment(lib, "d3d11.lib")
#pragma comment(lib, "dxgi.lib")

struct ViewfinderRenderer
{
    ID3D11Device* device = nullptr;
    ID3D11DeviceContext* context = nullptr;
    IDXGISwapChain1* swapChain = nullptr;
    ID3D11RenderTargetView* renderTargetView = nullptr;
};

static void ReleaseRendererResources(
    ViewfinderRenderer* renderer
)
{
    if (!renderer)
    {
        return;
    }

    if (renderer->renderTargetView)
    {
        renderer->renderTargetView->Release();
        renderer->renderTargetView = nullptr;
    }

    if (renderer->swapChain)
    {
        renderer->swapChain->Release();
        renderer->swapChain = nullptr;
    }

    if (renderer->context)
    {
        renderer->context->Release();
        renderer->context = nullptr;
    }

    if (renderer->device)
    {
        renderer->device->Release();
        renderer->device = nullptr;
    }
}

static HRESULT CreateD3D11Device(
    ID3D11Device** outDevice,
    ID3D11DeviceContext** outContext
)
{
    if (!outDevice || !outContext)
    {
        return E_INVALIDARG;
    }

    *outDevice = nullptr;
    *outContext = nullptr;

    D3D_FEATURE_LEVEL featureLevels[] =
    {
        D3D_FEATURE_LEVEL_11_1,
        D3D_FEATURE_LEVEL_11_0
    };

    D3D_FEATURE_LEVEL selectedFeatureLevel{};

    UINT flags =
        D3D11_CREATE_DEVICE_BGRA_SUPPORT;

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

static HRESULT CreateSwapChain(
    ID3D11Device* device,
    HWND hwnd,
    uint32_t width,
    uint32_t height,
    IDXGISwapChain1** outSwapChain
)
{
    if (!device || !hwnd || !outSwapChain)
    {
        return E_INVALIDARG;
    }

    *outSwapChain = nullptr;

    IDXGIDevice* dxgiDevice = nullptr;

    HRESULT hr = device->QueryInterface(
        __uuidof(IDXGIDevice),
        reinterpret_cast<void**>(&dxgiDevice)
    );

    if (FAILED(hr))
    {
        return hr;
    }

    IDXGIAdapter* adapter = nullptr;

    hr = dxgiDevice->GetAdapter(
        &adapter
    );

    if (FAILED(hr))
    {
        dxgiDevice->Release();
        return hr;
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
        return hr;
    }

    DXGI_SWAP_CHAIN_DESC1 desc{};

    desc.Width = width;
    desc.Height = height;

    desc.Format =
        DXGI_FORMAT_R8G8B8A8_UNORM;

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

    if (FAILED(hr))
    {
        return hr;
    }

    *outSwapChain = swapChain;

    return S_OK;
}

static HRESULT CreateRenderTarget(
    ID3D11Device* device,
    IDXGISwapChain1* swapChain,
    ID3D11RenderTargetView** outRenderTarget
)
{
    if (!device ||
        !swapChain ||
        !outRenderTarget)
    {
        return E_INVALIDARG;
    }

    *outRenderTarget = nullptr;

    ID3D11Texture2D* backBuffer = nullptr;

    HRESULT hr = swapChain->GetBuffer(
        0,
        __uuidof(ID3D11Texture2D),
        reinterpret_cast<void**>(&backBuffer)
    );

    if (FAILED(hr))
    {
        return hr;
    }

    ID3D11RenderTargetView* renderTarget = nullptr;

    hr = device->CreateRenderTargetView(
        backBuffer,
        nullptr,
        &renderTarget
    );

    backBuffer->Release();

    if (FAILED(hr))
    {
        return hr;
    }

    *outRenderTarget = renderTarget;

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderCreateRenderer(
    HWND hwnd,
    uint32_t width,
    uint32_t height,
    ViewfinderRenderer** outRenderer
)
{
    if (!hwnd || !outRenderer)
    {
        return E_INVALIDARG;
    }

    *outRenderer = nullptr;

    ViewfinderRenderer* renderer =
        new (std::nothrow) ViewfinderRenderer();

    if (!renderer)
    {
        return E_OUTOFMEMORY;
    }

    HRESULT hr = CreateD3D11Device(
        &renderer->device,
        &renderer->context
    );

    if (FAILED(hr))
    {
        ReleaseRendererResources(renderer);
        delete renderer;
        return static_cast<int32_t>(hr);
    }

    hr = CreateSwapChain(
        renderer->device,
        hwnd,
        width,
        height,
        &renderer->swapChain
    );

    if (FAILED(hr))
    {
        ReleaseRendererResources(renderer);
        delete renderer;
        return static_cast<int32_t>(hr);
    }

    hr = CreateRenderTarget(
        renderer->device,
        renderer->swapChain,
        &renderer->renderTargetView
    );

    if (FAILED(hr))
    {
        ReleaseRendererResources(renderer);
        delete renderer;
        return static_cast<int32_t>(hr);
    }

    *outRenderer = renderer;

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderRender(
    ViewfinderRenderer* renderer
)
{
    if (!renderer ||
        !renderer->device ||
        !renderer->context ||
        !renderer->swapChain ||
        !renderer->renderTargetView)
    {
        return E_INVALIDARG;
    }

    static float phase = 0.0f;

    phase += 0.005f;

    if (phase >= 1.0f)
    {
        phase = 0.0f;
    }

    const float clearColor[4] =
    {
        0.05f + phase * 0.15f,
        0.05f,
        0.08f + (1.0f - phase) * 0.15f,
        1.0f
    };

    renderer->context->OMSetRenderTargets(
        1,
        &renderer->renderTargetView,
        nullptr
    );

    renderer->context->ClearRenderTargetView(
        renderer->renderTargetView,
        clearColor
    );

    HRESULT hr = renderer->swapChain->Present(
        1,
        0
    );

    return static_cast<int32_t>(hr);
}

VIEWFINDER_API void ViewfinderDestroyRenderer(
    ViewfinderRenderer* renderer
)
{
    if (!renderer)
    {
        return;
    }

    ReleaseRendererResources(renderer);

    delete renderer;
}