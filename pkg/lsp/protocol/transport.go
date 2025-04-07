package protocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Transport struct {
	reader      io.Reader
	writer      io.Writer
	readMutex   sync.Mutex
	writeMutex  sync.Mutex
	headerBuf   bytes.Buffer
	contentBuf  bytes.Buffer
	scannerBuf  bytes.Buffer
	contentLen  int
	scannerInit bool
	closed      bool
	closeMutex  sync.Mutex
}

func NewTransport(reader io.Reader, writer io.Writer) *Transport {
	return &Transport{
		reader: reader,
		writer: writer,
		closed: false,
	}
}

func (t *Transport) IsClosed() bool {
	t.closeMutex.Lock()
	defer t.closeMutex.Unlock()
	return t.closed
}

func (t *Transport) Close() error {
	t.closeMutex.Lock()
	defer t.closeMutex.Unlock()

	if t.closed {
		return nil // Already closed
	}

	t.closed = true

	return nil
}

func (t *Transport) SendMessage(msg *JSONRPCMessage) error {
	t.writeMutex.Lock()
	defer t.writeMutex.Unlock()

	if t.IsClosed() {
		return fmt.Errorf("transport closed")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if _, err := fmt.Fprintf(t.writer, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		t.Close()
		return fmt.Errorf("failed to write header (transport closed): %w", err)
	}

	if _, err := t.writer.Write(data); err != nil {
		t.Close()
		return fmt.Errorf("failed to write content (transport closed): %w", err)
	}

	if f, ok := t.writer.(interface{ Flush() error }); ok {
		if err := f.Flush(); err != nil {
			t.Close()
			return fmt.Errorf("failed to flush writer (transport closed): %w", err)
		}
	}

	return nil
}

func (t *Transport) ReceiveMessage() (*JSONRPCMessage, error) {
	t.readMutex.Lock()
	defer t.readMutex.Unlock()

	if t.IsClosed() {
		return nil, fmt.Errorf("transport closed")
	}

	timeoutDuration := 30 * time.Second
	resultCh := make(chan struct {
		msg *JSONRPCMessage
		err error
	}, 1)

	go func() {
		for {
			contentLength, err := t.readHeader()
			if err != nil {
				if err == io.EOF || strings.Contains(err.Error(), "pipe") || strings.Contains(err.Error(), "connection") {
					t.Close()
					resultCh <- struct {
						msg *JSONRPCMessage
						err error
					}{nil, fmt.Errorf("error reading header (transport closed): %w", err)}
					return
				}
				resultCh <- struct {
					msg *JSONRPCMessage
					err error
				}{nil, fmt.Errorf("error reading header: %w", err)}
				return
			}

			content, err := t.readContent(contentLength)
			if err != nil {
				if err == io.EOF || strings.Contains(err.Error(), "pipe") || strings.Contains(err.Error(), "connection") {
					t.Close()
					resultCh <- struct {
						msg *JSONRPCMessage
						err error
					}{nil, fmt.Errorf("error reading content (transport closed): %w", err)}
					return
				}
				resultCh <- struct {
					msg *JSONRPCMessage
					err error
				}{nil, fmt.Errorf("error reading content: %w", err)}
				return
			}

			var msg JSONRPCMessage
			if err := json.Unmarshal(content, &msg); err != nil {
				resultCh <- struct {
					msg *JSONRPCMessage
					err error
				}{nil, fmt.Errorf("error deserializing JSON-RPC message: %w", err)}
				return
			}

			messageType := "response"
			if msg.ID == nil {
				messageType = "notification"
			}
			log.Printf("📥 %s message received: %s", messageType, string(content))

			if msg.ID == nil {
				log.Printf("⏭️ Ignoring notification: %s", msg.Method)
				continue
			}

			resultCh <- struct {
				msg *JSONRPCMessage
				err error
			}{&msg, nil}
			return
		}
	}()

	select {
	case result := <-resultCh:
		return result.msg, result.err
	case <-time.After(timeoutDuration):
		return nil, fmt.Errorf("timeout: no response received after %v seconds", timeoutDuration.Seconds())
	}
}

func (t *Transport) readHeader() (int, error) {
	t.headerBuf.Reset()
	s, ok := t.reader.(*bufio.Reader)
	if !ok {
		s = bufio.NewReader(t.reader)
	}

	for {
		line, err := s.ReadString('\n')
		if err != nil {
			return 0, fmt.Errorf("error reading header line: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		t.headerBuf.WriteString(line)
		t.headerBuf.WriteByte('\n')

		if strings.HasPrefix(line, "Content-Length:") {
			contentLenStr := strings.TrimSpace(line[len("Content-Length:"):])
			contentLen, err := strconv.Atoi(contentLenStr)
			if err != nil {
				return 0, fmt.Errorf("invalid Content-Length: %w", err)
			}
			t.contentLen = contentLen
		}
	}

	if t.contentLen == 0 {
		return 0, fmt.Errorf("missing Content-Length header")
	}

	return t.contentLen, nil
}

func (t *Transport) readContent(length int) ([]byte, error) {
	content := make([]byte, length)
	n, err := io.ReadFull(t.reader, content)
	if err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	if n != length {
		return nil, fmt.Errorf("incomplete content: expected %d bytes, got %d", length, n)
	}

	return content, nil
}
