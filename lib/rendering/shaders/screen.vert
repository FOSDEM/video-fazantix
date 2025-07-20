#version 400

out vec2 UV;
uniform uint stageData;

void main() {
    vec2 orientation=vec2(0.5, -0.5);
    if ((stageData & 1) != 0) {
        orientation.y = 0.5;
    }
    if ((stageData & 2) != 0) {
        orientation.x = -0.5;
    }

    // Generate a single triangle that fits the entire viewport and
    // give it UV coordinates that make 0.0-1.0 fall inside the viewport
    vec2 vertices[3]=vec2[3](vec2(-1,-1), vec2(3,-1), vec2(-1, 3));
    gl_Position = vec4(vertices[gl_VertexID],0,1);
    UV = orientation * gl_Position.xy + vec2(0.5);
}
