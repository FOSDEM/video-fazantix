package theatre

import (
	"fmt"

	"github.com/fosdem/fazantix/lib/sink/windowsink"
	"github.com/fosdem/fazantix/lib/source/pdfsource"
)

func (t *Theatre) MoveSequence(direction int) {
	if t.Sequence == nil || len(t.Sequence) == 0 {
		return
	}

	t.SequencePos += direction
	if t.SequencePos < 0 {
		t.SequencePos = 0
		return
	}
	if t.SequencePos > len(t.Sequence)-1 {
		t.SequencePos = len(t.Sequence) - 1
		return
	}

	step := t.Sequence[t.SequencePos]

	for source, page := range step.Page {
		ps := t.SourceByName(source).(*pdfsource.PdfSource)
		ps.SetPage(page, false)
	}

	if step.Scene != "" {
		transition := true
		if step.Transition != nil {
			transition = *step.Transition
		}
		stage := step.Sink
		if stage == "" {
			for sn, s := range t.Stages {
				if _, ok := s.Sink.(*windowsink.WindowSink); ok {
					fmt.Println("Sink", sn, "is a window sink")
					stage = sn
					break
				} else {
					fmt.Println("Sink", sn, "is wrong")
				}
			}
		}
		err := t.SetScene(stage, step.Scene, transition)
		if err != nil {
			fmt.Println("Could not set scene: ", err)
		}
	}
}
