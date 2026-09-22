struct VSInput
{
    uint vertexID : SV_VertexID;
};

struct VSOutput
{
    float4 position : SV_POSITION;
    float2 uv       : TEXCOORD0;
};

VSOutput main(VSInput input)
{
    VSOutput output;

    float2 positions[4] =
    {
        float2(-1.0,  1.0),
        float2( 1.0,  1.0),
        float2(-1.0, -1.0),
        float2( 1.0, -1.0)
    };

    float2 uvs[4] =
    {
        float2(0.0, 0.0),
        float2(1.0, 0.0),
        float2(0.0, 1.0),
        float2(1.0, 1.0)
    };

    output.position = float4(
        positions[input.vertexID],
        0.0,
        1.0
    );

    output.uv = uvs[input.vertexID];

    return output;
}