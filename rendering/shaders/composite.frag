#version 400

in vec2 UV;

out vec4 color;

uniform sampler2D tex[{{ .NumLayers }} * 3];
uniform vec4 layerPosition[{{ .NumLayers }}];
uniform vec4 layerData[{{ .NumLayers }}];

vec4 sampleLayerYUV422(int layer, vec4 dve, vec4 data) {
	vec2 tpos = (UV / dve.z) - (dve.xy / dve.zw);
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
	float border = data.y/(1/dve.w);
	if(a == 0 && tpos.x > 0-(border*dve.w*2) && tpos.x < 1.0+(border*dve.w*2) && tpos.y > 0-border && tpos.y < 1.0+border) {
		col = vec3(1,1,1);
		a = 1.0;
	}
	a *= data.x;
	return vec4(col.r, col.g, col.b, a);
}

vec4 sampleLayerRGB(int layer, vec4 dve, vec4 data) {
	vec4 col = texture(tex[layer*3], (UV / dve.z) - (dve.xy / dve.zw));
	float border = data.y*(1/dve.w);
	vec2 bpos = (UV / dve.z) - (dve.xy / (dve.zw));
	if(col.a == 0 && bpos.x > 0-(border*dve.w*2) && bpos.x < 1.0+(border*dve.w*2) && bpos.y > 0-border && bpos.y < 1.0+border) {
		col = vec4(1,1,1,1);
	}
	col.a *= data.x;
	return col;
}

void main() {
    vec4 composite;
    {{ range $i, $layer := .Layers }}
        vec4 layer_{{ $i }} = sampleLayer{{ $layer.Frames.FrameType.String }}({{ $i }}, layerPosition[{{ $i }}], layerData[{{ $i }}]);

        {{ if eq $i 0 }}
            composite = layer_{{ $i }};
        {{ else }}
            composite = mix(composite, layer_{{ $i }}, layer_{{ $i }}.a);
        {{ end }}
    {{ end }}

	color = composite;
}
