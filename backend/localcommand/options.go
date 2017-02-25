package localcommand

import (
	"syscall"
	"text/template"
)

type Option func(*LocalCommand)

func WithCloseSignal(signal syscall.Signal) Option {
	return func(lcmd *LocalCommand) {
		lcmd.closeSignal = signal
	}
}

func WithTitleTemplate(tmpl *template.Template) Option {
	return func(lcmd *LocalCommand) {
		lcmd.titleTemplate = tmpl
	}
}
