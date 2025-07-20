#version 400

in vec2 UV;

out vec4 color;

uniform sampler2D tex[{{ .NumSources }} * 3];
uniform vec4 layerPosition[{{ .NumSources }}];
uniform vec4 layerData[{{ .NumSources }}];

vec4 sampleLayerYUV422(vec2 uv, int layer, vec4 dve, vec4 data) {
	vec2 tpos = (uv / dve.z) - (dve.xy / dve.zw);
	float Y = texture(tex[layer*3], tpos).r;
	float Cb = texture(tex[layer*3+2], tpos).r - 0.5;
	float Cr = texture(tex[layer*3+1], tpos).r - 0.5;
	vec3 yuv = vec3(Y, Cr, Cb);
        mat3 colorMatrix = mat3(
                1,   0,       1.402,
                1,  -0.344,  -0.714,
                1,   1.772,   0);
	vec3 col = yuv * colorMatrix;
	float a = 1.0;
	if(tpos.x < 0 || tpos.x > 1.0) {
		a = 0.0;
	}
	if(tpos.y < 0 || tpos.y > 1.0) {
		a = 0.0;
	}
	a *= data.x;
	return vec4(col.r, col.g, col.b, a);
}

vec4 sampleLayerYUYV(vec2 uv, int layer, vec4 dve, vec4 data) {
	vec2 tpos = (uv / dve.z) - (dve.xy / dve.zw);
	vec2 uvpos = (uv / dve.z) - (dve.xy / dve.zw);
	vec4 src = texture(tex[layer*3], uvpos);
	int width = textureSize(tex[layer*3], 0).x;
	float Y = fract(uvpos.x * width) >= 0.5 ? src.b : src.r;
	float Cr = src.g - 0.5;
	float Cb = src.a - 0.5;
	vec3 yuv = vec3(Y, Cr, Cb);
        mat3 colorMatrix = mat3(
                1,   0,       1.402,
                1,  -0.344,  -0.714,
                1,   1.772,   0);
	vec3 col = yuv * colorMatrix;
	float a = 1.0;
	if(tpos.x < 0 || tpos.x > 1.0) {
		a = 0.0;
	}
	if(tpos.y < 0 || tpos.y > 1.0) {
		a = 0.0;
	}
	a *= data.x;
	return vec4(col.r, col.g, col.b, a);
}

vec4 sampleLayerRGBA(vec2 uv, int layer, vec4 dve, vec4 data) {
	vec4 col = texture(tex[layer*3], (uv / dve.z) - (dve.xy / dve.zw));
	col.a *= data.x;
	return col;
}

void main() {
    vec4 composite;
    {{ range $i, $source := .Sources }}
        vec4 layer_{{ $i }} = sampleLayer{{ $source.Frames.FrameType.String }}(UV, {{ $i }}, layerPosition[{{ $i }}], layerData[{{ $i }}]);

        {{ if eq $i 0 }}
            composite = layer_{{ $i }};
        {{ else }}
            composite = mix(composite, layer_{{ $i }}, layer_{{ $i }}.a);
        {{ end }}
    {{ end }}

	color = composite;
}
