#version 400

in vec2 UV;

out vec4 color;

uniform sampler2D tex[{{ .NumSources }} * 3];
uniform vec4 layerPosition[{{ .NumLayers }}];
uniform vec4 layerData[{{ .NumLayers }}];
uniform uint sourceIndices[{{ .NumLayers }}];
uniform uint sourceTypes[{{ .NumSources }}];

vec4 sampleLayerYUV422(vec2 uv, uint src_idx, vec4 dve, vec4 data) {
	vec2 tpos = (uv / dve.z) - (dve.xy / dve.zw);
	float Y = texture(tex[src_idx*3], tpos).r;
	float Cb = texture(tex[src_idx*3+2], tpos).r - 0.5;
	float Cr = texture(tex[src_idx*3+1], tpos).r - 0.5;
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

vec4 sampleLayerDebugBBox(vec2 uv, uint src_idx, vec4 dve, vec4 data) {
	vec2 tpos = (uv / dve.z) - (dve.xy / dve.zw);
	float a = 1.0;
	if(tpos.x < 0 || tpos.x > 1.0) {
		a = 0.0;
	}
	if(tpos.y < 0 || tpos.y > 1.0) {
		a = 0.0;
	}
	return vec4(0, 0.1 * src_idx * a, tpos.x * a, a);
}

vec4 sampleLayerYUYV(vec2 uv, uint src_idx, vec4 dve, vec4 data) {
	vec2 tpos = (uv / dve.z) - (dve.xy / dve.zw);
	vec2 uvpos = (uv / dve.z) - (dve.xy / dve.zw);
	vec4 src = texture(tex[src_idx*3], uvpos);
	int width = textureSize(tex[src_idx*3], 0).x;
	float fpix = fract(uvpos.x * width);
	float Y = fpix * src.b + (1.0-fpix) * src.r;
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

vec4 sampleLayerRGBA(vec2 uv, uint src_idx, vec4 dve, vec4 data) {
	vec4 col = texture(tex[src_idx*3], (uv / dve.z) - (dve.xy / dve.zw));
	col.a *= data.x;
	return col;
}

vec4 sampleLayerRGB(vec2 uv, uint src_idx, vec4 dve, vec4 data) {
	vec4 col = texture(tex[src_idx*3], (uv / dve.z) - (dve.xy / dve.zw));
	col.a = 1.0;
	return col;
}

vec4 sampleLayer(vec2 uv, uint src_idx, vec4 dve, vec4 data, uint srcType) {
	return sampleLayerDebugBBox(uv, src_idx, dve, data);
	if (srcType == 0) { // YUV422Frames
		return sampleLayerYUV422(uv, src_idx, dve, data);
	}
	if (srcType == 1) { // YUV422pFrames
		return sampleLayerYUV422(uv, src_idx, dve, data);
	}
	if (srcType == 2) { // RGBAFrames
		return sampleLayerRGBA(uv, src_idx, dve, data);
	}
	if (srcType == 3) { // RGBFrames
		return sampleLayerRGB(uv, src_idx, dve, data);
	}

	return vec4(0, 0, 0, 0);
}

void main() {
    vec4 composite;
    {{ range $i := .NumLayers }}
        vec4 layer_{{ $i }} = sampleLayer(
			UV,
			sourceIndices[{{ $i }}],
			layerPosition[{{ $i }}],
			layerData[{{ $i }}],
			sourceTypes[sourceIndices[{{ $i }}]]
		);

        {{ if eq $i 0 }}
            composite = layer_{{ $i }};
        {{ else }}
            composite = mix(composite, layer_{{ $i }}, layer_{{ $i }}.a);
        {{ end }}
    {{ end }}

	color = composite;
}
