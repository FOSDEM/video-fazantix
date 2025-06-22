#version 400

in vec2 position;
in vec2 uv;

out vec2 UV;

void main() {
    gl_Position = vec4(position, 0.0, 1.0);
    UV = uv;
}
