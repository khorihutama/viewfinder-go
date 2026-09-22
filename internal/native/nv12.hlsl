Texture2D<float> YTexture : register(t0);
Texture2D<float2> UVTexture : register(t1);

SamplerState LinearSampler : register(s0);

struct PSInput
{
    float4 position : SV_POSITION;
    float2 uv       : TEXCOORD0;
};

float4 main(PSInput input) : SV_TARGET
{
    float y = YTexture.Sample(
        LinearSampler,
        input.uv
    );

    float2 uv = UVTexture.Sample(
        LinearSampler,
        input.uv
    );

    float Y = y;
    float U = uv.x;
    float V = uv.y;

    // BT.601 video-range NV12 -> RGB

    Y = 1.16438356 * (Y - 0.0625);
    U = U - 0.5;
    V = V - 0.5;

    float R = Y + 1.596027 * V;
    float G = Y - 0.391762 * U - 0.812968 * V;
    float B = Y + 2.017232 * U;

    return float4(
        saturate(R),
        saturate(G),
        saturate(B),
        1.0
    );
}