package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const maxMessageSize = 1 << 20

// Handler handles one validated JSON-RPC request.
type Handler func(context.Context, Request) (any, *Error)

// ServeConn serves newline-delimited JSON-RPC requests on one connection.
func ServeConn(ctx context.Context, connection io.ReadWriteCloser, handler Handler) error {
	defer connection.Close()

	reader := bufio.NewScanner(connection)
	reader.Buffer(make([]byte, 4<<10), maxMessageSize)
	writer := bufio.NewWriter(connection)

	for reader.Scan() {
		response := handle(ctx, reader.Bytes(), handler)
		if err := writeResponse(writer, response); err != nil {
			return err
		}
	}
	if err := reader.Err(); err != nil {
		return err
	}
	return nil
}

func handle(ctx context.Context, rawRequest []byte, handler Handler) Response {
	var request Request
	if err := json.Unmarshal(rawRequest, &request); err != nil {
		return errorResponse(nil, CodeParseError, "parse error")
	}
	if request.JSONRPC != Version || len(request.ID) == 0 || request.Method == "" {
		return errorResponse(request.ID, CodeInvalidRequest, "invalid request")
	}

	result, rpcError := handler(ctx, request)
	if rpcError != nil {
		return Response{JSONRPC: Version, ID: request.ID, Error: rpcError}
	}
	return Response{JSONRPC: Version, ID: request.ID, Result: result}
}

func errorResponse(id json.RawMessage, code int, message string) Response {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return Response{
		JSONRPC: Version,
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
}

func writeResponse(writer *bufio.Writer, response Response) error {
	encoded, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	if _, err := writer.Write(append(encoded, '\n')); err != nil {
		return err
	}
	return writer.Flush()
}

// MethodNotFound returns the standard error for unsupported methods.
func MethodNotFound(method string) *Error {
	return &Error{Code: CodeMethodNotFound, Message: "method not found: " + method}
}

// InternalError converts an unexpected error into a safe protocol error.
func InternalError(err error) *Error {
	if errors.Is(err, context.Canceled) {
		return &Error{Code: CodeInternalError, Message: "request canceled"}
	}
	return &Error{Code: CodeInternalError, Message: "internal error"}
}
