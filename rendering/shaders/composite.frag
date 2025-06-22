#version 400

in vec2 UV;

out vec4 color;

uniform sampler2D tex[9];
uniform vec4 sourcePosition[3];

vec4 sampleLayerYUV(int layer, vec4 dve) {
	vec2 tpos = (UV / dve.z) - (dve.xy / dve.z);
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
	return vec4(col.r, col.g, col.b, a);
}

vec4 sampleLayerRGB(int layer, vec4 dve) {
	return texture(tex[layer*3], (UV / dve.z) - (dve.xy / dve.z));
}

void main() {
	vec4 background = sampleLayerRGB(0, sourcePosition[0]);
	// vec4 background = sampleLayerYUV(0, sourcePosition[0]);
	vec4 dve1 = sampleLayerRGB(1, sourcePosition[1]);
	vec4 dve2 = sampleLayerRGB(2, sourcePosition[2]);
	vec4 temp = mix(background, dve1, dve1.a);
	color = mix(temp, dve2, dve2.a);
}
