#### low-latency framedropping mode:
- GetFrameForReading should:
    - quickly return the current reading frame
    - increment its readers field
    - sets its "ever read" flag to true

- FinishedReading should:
    - decrement the frame's readers field
    - if it is zero, move the frame to the recycle bin

- GetFrameForWriting should:
    - quickly try getting a frame from the recycle bin
    - if no frame was available, create a frame by depleting the frame budget
    - if the frame budget is exhausted, it should return nil and allow the caller to framedrop
        - log a framedrop
    - frame budget never gets increased
    - reset the frame's "ever read" flag

- FinishedWriting should:
    - see if the current reading frame has ever been read
        - if not, log a framedrop and move it to the recycle bin
    - replace the current reading frame pointer

#### buffered mode (for recording, etc):
- GetFrameForReading should:
    - pop the frame queue (blocking)
- FinishedReading should:
    - push to the recycle bin
- GetFrameForWriting should:
    - quickly try getting a frame from the recycle bin
    - if no frame was available, create a frame by depleting the frame budget
    - if the frame budget is exhausted, it should return nil and allow the caller to framedrop
        - log a framedrop
    - frame budget never gets increased
    - reset the frame's "ever read" flag
- FinishedWriting should:
    - push to the frame queue

#### misc
* the frame queue size should be the same as the frame budget, so that writes never block
* all of these should be atomic
