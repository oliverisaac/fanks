package types

import "fmt"

type FormData struct {
	Config Config
	Error  error
}

func NewFormData(cfg Config) *FormData {
	return &FormData{
		Config: cfg,
	}
}

func (fd *FormData) WithErrorf(format string, args ...any) *FormData {
	return fd.WithError(fmt.Errorf(format, args...))
}

func (fd *FormData) WithError(err error) *FormData {
	fd.Error = err
	return fd
}
