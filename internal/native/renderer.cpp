#include "viewfinder_native.h"

#include <windows.h>
#include <d3d11.h>
#include <d3dcompiler.h>
#include <dxgi.h>

#include <string>

#pragma comment(lib, "d3d11.lib")
#pragma comment(lib, "dxgi.lib")
#pragma comment(lib, "d3dcompiler.lib")

static ID3D11Device* g_device = nullptr;
static ID3D11DeviceContext* g_context = nullptr;
static IDXGISwapChain* g_swapChain = nullptr;
static ID3D11RenderTargetView* g_renderTarget = nullptr;

static ID3D11Texture2D* g_yTexture = nullptr;
static ID3D11Texture2D* g_uvTexture = nullptr;

static ID3D11ShaderResourceView* g_yView = nullptr;
static ID3D11ShaderResourceView* g_uvView = nullptr;

static ID3D11VertexShader* g_vertexShader = nullptr;
static ID3D11PixelShader* g_pixelShader = nullptr;

static ID3D11SamplerState* g_sampler = nullptr;
static uint32_t g_backBufferWidth = 0;
static uint32_t g_backBufferHeight = 0;
static uint32_t g_videoWidth = 0;
static uint32_t g_videoHeight = 0;
static bool g_fillMode = false;

static void SetViewport()
{
    if (!g_context || !g_backBufferWidth || !g_backBufferHeight)
    {
        return;
    }

    float width = static_cast<float>(g_backBufferWidth);
    float height = static_cast<float>(g_backBufferHeight);

    if (g_videoWidth && g_videoHeight)
    {
        const float videoAspect =
            static_cast<float>(g_videoWidth) / g_videoHeight;
        const float windowAspect = width / height;

        if (!g_fillMode && videoAspect > windowAspect)
        {
            height = width / videoAspect;
        }
        else if (!g_fillMode)
        {
            width = height * videoAspect;
        }
        else if (videoAspect > windowAspect)
        {
            width = height * videoAspect;
        }
        else
        {
            height = width / videoAspect;
        }
    }

    D3D11_VIEWPORT viewport{};
    viewport.TopLeftX = (g_backBufferWidth - width) / 2.0f;
    viewport.TopLeftY = (g_backBufferHeight - height) / 2.0f;
    viewport.Width = width;
    viewport.Height = height;
    viewport.MinDepth = 0.0f;
    viewport.MaxDepth = 1.0f;
    g_context->RSSetViewports(1, &viewport);
}

VIEWFINDER_API int32_t ViewfinderRendererSetFillMode(int32_t fill)
{
    g_fillMode = fill != 0;
    SetViewport();
    return S_OK;
}

static HRESULT CreateRenderTarget()
{
    ID3D11Texture2D* backBuffer = nullptr;
    HRESULT hr = g_swapChain->GetBuffer(
        0,
        __uuidof(ID3D11Texture2D),
        reinterpret_cast<void**>(&backBuffer)
    );

    if (FAILED(hr))
    {
        return hr;
    }

    hr = g_device->CreateRenderTargetView(backBuffer, nullptr, &g_renderTarget);
    backBuffer->Release();
    return hr;
}

static HRESULT CompileShader(
    const wchar_t* path,
    const char* profile,
    ID3DBlob** outBlob
)
{
    ID3DBlob* errors = nullptr;
    HRESULT hr = D3DCompileFromFile(
        path,
        nullptr,
        D3D_COMPILE_STANDARD_FILE_INCLUDE,
        "main",
        profile,
        0,
        0,
        outBlob,
        &errors
    );

    if (hr == HRESULT_FROM_WIN32(ERROR_FILE_NOT_FOUND) ||
        hr == HRESULT_FROM_WIN32(ERROR_PATH_NOT_FOUND))
    {
        if (errors)
        {
            errors->Release();
            errors = nullptr;
        }

        std::wstring fallbackPath = L"..\\";
        fallbackPath += path;

        hr = D3DCompileFromFile(
            fallbackPath.c_str(),
            nullptr,
            D3D_COMPILE_STANDARD_FILE_INCLUDE,
            "main",
            profile,
            0,
            0,
            outBlob,
            &errors
        );
    }

    if (errors)
    {
        errors->Release();
    }

    return hr;
}

static HRESULT CreateShaders()
{
    ID3DBlob* vertexBlob = nullptr;
    HRESULT hr = CompileShader(
        L"internal\\native\\fullscreen.hlsl",
        "vs_5_0",
        &vertexBlob
    );

    if (FAILED(hr))
    {
        return hr;
    }

    hr = g_device->CreateVertexShader(
        vertexBlob->GetBufferPointer(),
        vertexBlob->GetBufferSize(),
        nullptr,
        &g_vertexShader
    );
    vertexBlob->Release();

    if (FAILED(hr))
    {
        return hr;
    }

    ID3DBlob* pixelBlob = nullptr;
    hr = CompileShader(
        L"internal\\native\\nv12.hlsl",
        "ps_5_0",
        &pixelBlob
    );

    if (FAILED(hr))
    {
        return hr;
    }

    hr = g_device->CreatePixelShader(
        pixelBlob->GetBufferPointer(),
        pixelBlob->GetBufferSize(),
        nullptr,
        &g_pixelShader
    );
    pixelBlob->Release();

    if (FAILED(hr))
    {
        return hr;
    }

    D3D11_SAMPLER_DESC samplerDesc{};
    samplerDesc.Filter = D3D11_FILTER_MIN_MAG_MIP_LINEAR;
    samplerDesc.AddressU = D3D11_TEXTURE_ADDRESS_CLAMP;
    samplerDesc.AddressV = D3D11_TEXTURE_ADDRESS_CLAMP;
    samplerDesc.AddressW = D3D11_TEXTURE_ADDRESS_CLAMP;
    samplerDesc.MaxLOD = D3D11_FLOAT32_MAX;

    return g_device->CreateSamplerState(&samplerDesc, &g_sampler);
}

static void ReleaseRenderer()
{
    g_backBufferWidth = 0;
    g_backBufferHeight = 0;
    g_videoWidth = 0;
    g_videoHeight = 0;

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

    if (g_yView)
    {
        g_yView->Release();
        g_yView = nullptr;
    }

    if (g_uvView)
    {
        g_uvView->Release();
        g_uvView = nullptr;
    }

    if (g_yTexture)
    {
        g_yTexture->Release();
        g_yTexture = nullptr;
    }

    if (g_uvTexture)
    {
        g_uvTexture->Release();
        g_uvTexture = nullptr;
    }

    if (g_vertexShader)
    {
        g_vertexShader->Release();
        g_vertexShader = nullptr;
    }

    if (g_pixelShader)
    {
        g_pixelShader->Release();
        g_pixelShader = nullptr;
    }

    if (g_sampler)
    {
        g_sampler->Release();
        g_sampler = nullptr;
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

    hr = CreateRenderTarget();

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

    g_backBufferWidth = width;
    g_backBufferHeight = height;
    SetViewport();

    hr = CreateShaders();
    if (FAILED(hr))
    {
        ReleaseRenderer();
        return static_cast<int32_t>(hr);
    }

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

VIEWFINDER_API int32_t ViewfinderRendererResize(
    uint32_t width,
    uint32_t height
)
{
    if (!g_device || !g_context || !g_swapChain || !g_renderTarget ||
        !width || !height)
    {
        return E_INVALIDARG;
    }

    g_context->OMSetRenderTargets(0, nullptr, nullptr);
    g_renderTarget->Release();
    g_renderTarget = nullptr;

    HRESULT hr = g_swapChain->ResizeBuffers(
        0,
        width,
        height,
        DXGI_FORMAT_UNKNOWN,
        0
    );
    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    hr = CreateRenderTarget();
    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    g_backBufferWidth = width;
    g_backBufferHeight = height;
    SetViewport();
    return S_OK;
}

VIEWFINDER_API void ViewfinderRendererDestroy()
{
    ReleaseRenderer();
}

static HRESULT CreateNV12Textures(
    uint32_t width,
    uint32_t height
)
{
    if (!g_device)
    {
        return E_FAIL;
    }

    if (g_yTexture)
    {
        g_yTexture->Release();
        g_yTexture = nullptr;
    }

    if (g_uvTexture)
    {
        g_uvTexture->Release();
        g_uvTexture = nullptr;
    }

    if (g_yView)
    {
        g_yView->Release();
        g_yView = nullptr;
    }

    if (g_uvView)
    {
        g_uvView->Release();
        g_uvView = nullptr;
    }

    // ---------------------------------------------------------
    // Y plane
    // ---------------------------------------------------------

    D3D11_TEXTURE2D_DESC yDesc{};

    yDesc.Width = width;
    yDesc.Height = height;
    yDesc.MipLevels = 1;
    yDesc.ArraySize = 1;

    yDesc.Format = DXGI_FORMAT_R8_UNORM;

    yDesc.SampleDesc.Count = 1;

    yDesc.Usage = D3D11_USAGE_DYNAMIC;

    yDesc.BindFlags =
        D3D11_BIND_SHADER_RESOURCE;

    yDesc.CPUAccessFlags =
        D3D11_CPU_ACCESS_WRITE;

    HRESULT hr = g_device->CreateTexture2D(
        &yDesc,
        nullptr,
        &g_yTexture
    );

    if (FAILED(hr))
    {
        return hr;
    }

    // ---------------------------------------------------------
    // UV plane
    // ---------------------------------------------------------

    D3D11_TEXTURE2D_DESC uvDesc{};

    uvDesc.Width = width / 2;
    uvDesc.Height = height / 2;

    uvDesc.MipLevels = 1;
    uvDesc.ArraySize = 1;

    uvDesc.Format = DXGI_FORMAT_R8G8_UNORM;

    uvDesc.SampleDesc.Count = 1;

    uvDesc.Usage = D3D11_USAGE_DYNAMIC;

    uvDesc.BindFlags =
        D3D11_BIND_SHADER_RESOURCE;

    uvDesc.CPUAccessFlags =
        D3D11_CPU_ACCESS_WRITE;

    hr = g_device->CreateTexture2D(
        &uvDesc,
        nullptr,
        &g_uvTexture
    );

    if (FAILED(hr))
    {
        return hr;
    }

    // ---------------------------------------------------------
    // Y shader resource view
    // ---------------------------------------------------------

    D3D11_SHADER_RESOURCE_VIEW_DESC yViewDesc{};

    yViewDesc.Format =
        DXGI_FORMAT_R8_UNORM;

    yViewDesc.ViewDimension =
        D3D11_SRV_DIMENSION_TEXTURE2D;

    yViewDesc.Texture2D.MipLevels = 1;

    hr = g_device->CreateShaderResourceView(
        g_yTexture,
        &yViewDesc,
        &g_yView
    );

    if (FAILED(hr))
    {
        return hr;
    }

    // ---------------------------------------------------------
    // UV shader resource view
    // ---------------------------------------------------------

    D3D11_SHADER_RESOURCE_VIEW_DESC uvViewDesc{};

    uvViewDesc.Format =
        DXGI_FORMAT_R8G8_UNORM;

    uvViewDesc.ViewDimension =
        D3D11_SRV_DIMENSION_TEXTURE2D;

    uvViewDesc.Texture2D.MipLevels = 1;

    hr = g_device->CreateShaderResourceView(
        g_uvTexture,
        &uvViewDesc,
        &g_uvView
    );

    if (FAILED(hr))
    {
        return hr;
    }

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderRendererUploadNV12(
    const uint8_t* data,
    uint32_t dataSize,
    uint32_t width,
    uint32_t height,
    uint32_t stride
)
{
    if (!data || width == 0 || height == 0 ||
        width % 2 != 0 || height % 2 != 0 || stride < width)
    {
        return E_INVALIDARG;
    }

    if (!g_device || !g_context)
    {
        return E_FAIL;
    }

    const uint64_t expectedSize =
        static_cast<uint64_t>(stride) * height * 3 / 2;

    if (dataSize < expectedSize)
    {
        return E_INVALIDARG;
    }

    static uint32_t textureWidth = 0;
    static uint32_t textureHeight = 0;

    if (!g_yTexture ||
        textureWidth != width ||
        textureHeight != height)
    {
        HRESULT hr = CreateNV12Textures(
            width,
            height
        );

        if (FAILED(hr))
        {
            return static_cast<int32_t>(hr);
        }

        textureWidth = width;
        textureHeight = height;
        g_videoWidth = width;
        g_videoHeight = height;
        SetViewport();
    }

    // ---------------------------------------------------------
    // Y plane
    // ---------------------------------------------------------

    D3D11_MAPPED_SUBRESOURCE mapped{};

    HRESULT hr = g_context->Map(
        g_yTexture,
        0,
        D3D11_MAP_WRITE_DISCARD,
        0,
        &mapped
    );

    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    const uint8_t* srcY = data;

    uint8_t* dstY =
        static_cast<uint8_t*>(mapped.pData);

    for (uint32_t y = 0; y < height; y++)
    {
        memcpy(
            dstY + y * mapped.RowPitch,
            srcY + y * stride,
            width
        );
    }

    g_context->Unmap(
        g_yTexture,
        0
    );

    // ---------------------------------------------------------
    // UV plane
    // ---------------------------------------------------------

    hr = g_context->Map(
        g_uvTexture,
        0,
        D3D11_MAP_WRITE_DISCARD,
        0,
        &mapped
    );

    if (FAILED(hr))
    {
        return static_cast<int32_t>(hr);
    }

    const uint8_t* srcUV =
        data + stride * height;

    uint8_t* dstUV =
        static_cast<uint8_t*>(mapped.pData);

    const uint32_t uvHeight =
        height / 2;

    const uint32_t uvWidth =
        width;

    for (uint32_t y = 0; y < uvHeight; y++)
    {
        memcpy(
            dstUV + y * mapped.RowPitch,
            srcUV + y * stride,
            uvWidth
        );
    }

    g_context->Unmap(
        g_uvTexture,
        0
    );

    return S_OK;
}

VIEWFINDER_API int32_t ViewfinderRendererDraw()
{
    if (!g_context || !g_renderTarget || !g_yView || !g_uvView ||
        !g_vertexShader || !g_pixelShader || !g_sampler)
    {
        return E_FAIL;
    }

    ID3D11ShaderResourceView* views[] = {g_yView, g_uvView};
    const float black[] = {0.0f, 0.0f, 0.0f, 1.0f};

    g_context->OMSetRenderTargets(1, &g_renderTarget, nullptr);
    g_context->ClearRenderTargetView(g_renderTarget, black);
    g_context->IASetInputLayout(nullptr);
    g_context->IASetPrimitiveTopology(D3D11_PRIMITIVE_TOPOLOGY_TRIANGLESTRIP);
    g_context->VSSetShader(g_vertexShader, nullptr, 0);
    g_context->PSSetShader(g_pixelShader, nullptr, 0);
    g_context->PSSetShaderResources(0, 2, views);
    g_context->PSSetSamplers(0, 1, &g_sampler);
    g_context->Draw(4, 0);

    ID3D11ShaderResourceView* emptyViews[] = {nullptr, nullptr};
    g_context->PSSetShaderResources(0, 2, emptyViews);

    return S_OK;
}
