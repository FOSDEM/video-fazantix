package layer

type Source interface {
	Frames() *FrameForwarder
	Start() bool
}
