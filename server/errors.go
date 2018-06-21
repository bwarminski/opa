package server

import (
	"github.com/open-policy-agent/opa/server/types"
	"github.com/open-policy-agent/opa/ast"
	"fmt"
)

const (
	ModuleParseErr = "module_parse_error"
	EntryExistsErr = "entry_exists"
)

type Error struct {
	Code string
	Message string
}

func (err *Error) Error() string {
	return err.Message
}

type WrappedError struct {
	Code string
	Nested error
}

func (err *WrappedError) Error() string {
	return err.Nested.Error()
}

func NewWrappedError(code string, err error) error {
	return &WrappedError{
		Code: code,
		Nested: err,
	}
}

type CompileError struct {
	Errors []*ast.Error
}

func (err *CompileError) Error() string {
	return fmtAstError(types.MsgCompileModuleError, err.Errors)
}

func NewCompileError(errors []*ast.Error) error {
	return &CompileError{
		Errors: errors,
	}
}

func fmtAstError(f string, a ...interface{}) string {
	return fmt.Sprintf(f, a...)
}

func IsEntryExistsError(err error) bool {
	switch err := err.(type) {
	case *Error:
		if err.Code == EntryExistsErr {
			return true
		}
		return false
	}

	return false
}

func IsModuleParseError(err error) bool {
	switch err := err.(type) {
	case *WrappedError:
		if err.Code == ModuleParseErr {
			return true
		}
		return false
	}

	return false
}

