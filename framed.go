/*
Package framed provides an implementations of io.Writer and io.Reader that write
and read whole frames only.

Frames are length-prefixed.  The first two bytes are an unsigned 16 bit int
stored in little-endian byte order indicating the length of the content.  The
remaining bytes are the actual content of the frame.

The use of a uint16 means that the maximum possible frame size (MaxFrameSize)
is 65535.
*/
package framed

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

const (
	// FrameHeaderBits is the size of the frame header in bits
	FrameHeaderBits = 16

	// FrameHeaderLength is the size of the frame header in bytes
	FrameHeaderLength = FrameHeaderBits / 8

	// MaxFrameLength is the maximum possible size of a frame (not including the
	// length prefix)
	MaxFrameLength = 1<<FrameHeaderBits - 1

	tooLongError = "Attempted to write frame of length %d which is longer than maximum allowed length of %d"
)

var endianness = binary.LittleEndian

/*
A Reader enhances an io.ReadCloser to read data in contiguous frames. It
implements the io.Reader interface, but unlike typical io.Readers it only
returns whole frames.

A Reader also supports the ability to read frames using dynamically allocated
buffers via the ReadFrame method.
*/
type Reader struct {
	Stream io.Reader // the raw underlying connection
	mutex  sync.Mutex
}

/*
A Writer enhances an io.WriteCLoser to write data in contiguous frames. It
implements the io.Writer interface, but unlike typical io.Writers, it includes
information that allows a corresponding Reader to read whole frames without them
being fragmented.

A Writer also supports a method that writes multiple buffers to the underlying
stream as a single frame.
*/
type Writer struct {
	Stream io.Writer // the raw underlying connection
	mutex  sync.Mutex
}

/*
ReadWriteCloser combines a Reader and a Writer on top of an underlying
ReadWriteCloser.
*/
type ReadWriteCloser struct {
	Reader
	Writer
	io.Closer
}

func NewReader(r io.Reader) *Reader {
	return &Reader{Stream: r}
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{Stream: w}
}

func NewReadWriteCloser(rwc io.ReadWriteCloser) *ReadWriteCloser {
	return &ReadWriteCloser{Reader{Stream: rwc}, Writer{Stream: rwc}, rwc}
}

/*
Read implements the function from io.Reader.  Unlike io.Reader.Read,
frame.Read only returns full frames of data (assuming that the data was written
by a framed.Writer).
*/
func (framed *Reader) Read(buffer []byte) (n int, err error) {
	framed.mutex.Lock()
	defer framed.mutex.Unlock()

	var nb uint16
	innererr = binary.Read(framed.Stream, endianness, &nb)
	if innererr != nil {
		return
	}

	innern = int(nb)

	bufferSize := len(buffer)
	if innern > bufferSize {
		return 0, fmt.Errorf("Buffer of size %d is too small to hold frame of size %d", bufferSize, n)
	}

	// Read into buffer
	innern, innererr = io.ReadFull(framed.Stream, buffer[:innern])
	return
}

// ReadFrame reads the next frame, using a new buffer sized to hold the frame.
func (framed *Reader) ReadFrame() (frame []byte, err error) {
	framed.mutex.Lock()
	defer framed.mutex.Unlock()

	var nb uint16
	innererr = binary.Read(framed.Stream, endianness, &nb)
	if innererr != nil {
		return
	}

	l := int(nb)
	innerframe = make([]byte, l)

	// Read into buffer
	_, innererr = io.ReadFull(framed.Stream, innerframe)
	return
}

/*
Write implements the Write method from io.Writer.  It prepends a frame length
header that allows the framed.Reader on the other end to read the whole frame.
*/
func (framed *Writer) Write(frame []byte) (n int, err error) {
	framed.mutex.Lock()
	defer framed.mutex.Unlock()

	innern = len(frame)
	if innern > MaxFrameLength {
		return 0, fmt.Errorf(tooLongError, n, MaxFrameLength)
	}

	// Write the length header
	if innererr = binary.Write(framed.Stream, endianness, uint16(innern)); innererr != nil {
		return
	}

	// Write the data
	var written int
	if written, innererr = framed.Stream.Write(frame); innererr != nil {
		return
	}
	if written != innern {
		innererr = fmt.Errorf("%d bytes written, expected to write %d", written, n)
	}
	return
}

func (framed *Writer) WritePieces(pieces ...[]byte) (n int, err error) {
	framed.mutex.Lock()
	defer framed.mutex.Unlock()

	for _, piece := range pieces {
		innern = innern + len(piece)
	}

	if n > MaxFrameLength {
		return 0, fmt.Errorf(tooLongError, n, MaxFrameLength)
	}

	// Write the length header
	if innererr = binary.Write(framed.Stream, endianness, uint16(n)); innererr != nil {
		return
	}

	// Write the data
	var written int
	for _, piece := range pieces {
		var nw int
		if nw, innererr = framed.Stream.Write(piece); innererr != nil {
			return
		}
		written = written + nw
	}
	if written != n {
		innererr = fmt.Errorf("%d bytes written, expected to write %d", written, n)
	}
	return
}
