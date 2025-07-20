#version 400

out vec2 UV;

void main() {
    // Generate a single triangle that fits the entire viewport and
    // give it UV coordinates that make 0.0-1.0 fall inside the viewport
    vec2 vertices[3]=vec2[3](vec2(-1,-1), vec2(3,-1), vec2(-1, 3));
    gl_Position = vec4(vertices[gl_VertexID],0,1);
    UV = vec2(0.5, -0.5) * gl_Position.xy + vec2(0.5);
}
