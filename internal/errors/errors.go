package errors

import (
	"fmt"
	"runtime"
)

type VaultSyncError struct {
	Op      string
	Path    string
	Err     error
	Context map[string]interface{}
	File    string
	Line    int
}

func (e *VaultSyncError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *VaultSyncError) Unwrap() error {
	return e.Err
}

func New(op string, err error) *VaultSyncError {
	_, file, line, _ := runtime.Caller(1)
	return &VaultSyncError{
		Op:      op,
		Err:     err,
		Context: make(map[string]interface{}),
		File:    file,
		Line:    line,
	}
}

func NewWithPath(op, path string, err error) *VaultSyncError {
	_, file, line, _ := runtime.Caller(1)
	return &VaultSyncError{
		Op:      op,
		Path:    path,
		Err:     err,
		Context: make(map[string]interface{}),
		File:    file,
		Line:    line,
	}
}

func (e *VaultSyncError) WithContext(key string, value interface{}) *VaultSyncError {
	e.Context[key] = value
	return e
}

func Wrap(err error, op string) error {
	if err == nil {
		return nil
	}
	
	if vaultErr, ok := err.(*VaultSyncError); ok {
		return vaultErr
	}
	
	_, file, line, _ := runtime.Caller(1)
	return &VaultSyncError{
		Op:      op,
		Err:     err,
		Context: make(map[string]interface{}),
		File:    file,
		Line:    line,
	}
}

func WrapWithPath(err error, op, path string) error {
	if err == nil {
		return nil
	}
	
	if vaultErr, ok := err.(*VaultSyncError); ok {
		if vaultErr.Path == "" {
			vaultErr.Path = path
		}
		return vaultErr
	}
	
	_, file, line, _ := runtime.Caller(1)
	return &VaultSyncError{
		Op:      op,
		Path:    path,
		Err:     err,
		Context: make(map[string]interface{}),
		File:    file,
		Line:    line,
	}
}