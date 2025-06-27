#### at startup:
- each frame forwarder has a number N, the number of buffered frames
- allocate N black frames using the PBO allocator
- prefill the recycle bin with N-1 of these frames and use the other one as a "reading frame"

#### low-latency framedropping mode:
- GetFrameForReading should:
    - quickly return the current reading frame
    - increment its readers field

- FinishedReading should:
    - decrement the frame's readers field
    - if it is zero and the frame is marked for recycling, move it to the recycle bin

- GetFrameForWriting should:
    - try getting a frame from the recycle bin
    - if no frame was available return nil and allow the caller to framedrop
        - log a framedrop
    - update the frame's identifier (random number, may be consecutive - used in the render loop to test if a new frame was produced)
    - unmark the frame for recycling and return it

- FinishedWriting should:
    - if the current reading frame's readers field is nonzero, mark the frame for recycling
    - otherwise, instantly recycle it
    - replace the current reading frame pointer with the given frame

#### buffered mode (for recording, etc):
- GetFrameForReading should:
    - pop the frame queue (blocking)
- FinishedReading should:
    - push to the recycle bin
- GetFrameForWriting should:
    - try getting a frame from the recycle bin
    - if no frame was available return nil and allow the caller to framedrop
        - log a framedrop
    - unmark the frame for recycling and return it
- FinishedWriting should:
    - push to the frame queue

#### misc
* the frame queue size should be N, so that writes never block
* all of these operations should be atomic (one mutex for the frame forwarder for all operations, and also handle the number of readers field with atomic intrinsics
* frame budget never gets increased
