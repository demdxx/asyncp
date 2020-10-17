package asyncp

import "context"

type (
	// ContextWrapperFnk for prepare execution context
	ContextWrapperFnk func(ctx context.Context) context.Context
	// PanicHandlerFnk for any panic errors
	PanicHandlerFnk func(Task, Event, interface{})
	// ErrorHandlerFnk for any error response
	ErrorHandlerFnk func(Task, Event, error)
)

// Options of the mux server
type Options struct {
	MainExecContext context.Context
	PanicHandler    PanicHandlerFnk
	ErrorHandler    ErrorHandlerFnk
	ContextWrapper  ContextWrapperFnk
	ResponseFactory ResponseWriterFactory
}

// Option of the task configuration
type Option func(opt *Options)

// WithMainExecContext puts main execution context to the Mux option
func WithMainExecContext(ctx context.Context) Option {
	return func(opt *Options) {
		opt.MainExecContext = ctx
	}
}

// WithPanicHandler puts panic handler to the Mux option
func WithPanicHandler(h PanicHandlerFnk) Option {
	return func(opt *Options) {
		opt.PanicHandler = h
	}
}

// WithErrorHandler puts error handler to the Mux option
func WithErrorHandler(h ErrorHandlerFnk) Option {
	return func(opt *Options) {
		opt.ErrorHandler = h
	}
}

// WithContextWrapper puts context wrapper to the Mux option
func WithContextWrapper(w ContextWrapperFnk) Option {
	return func(opt *Options) {
		opt.ContextWrapper = w
	}
}

// WithStreamResponseMap set option with stream mapping converted to factory
func WithStreamResponseMap(streams ...interface{}) Option {
	return func(opt *Options) {
		opt.ResponseFactory = NewMultistreamResponseFactory(streams...)
	}
}

// WithResponseFactory set option with stream factory
func WithResponseFactory(responseFactory ResponseWriterFactory) Option {
	return func(opt *Options) {
		opt.ResponseFactory = responseFactory
	}
}

// WithStreamResponsePublisher set option with single stream publisher
func WithStreamResponsePublisher(publisher Publisher) Option {
	return func(opt *Options) {
		opt.ResponseFactory = NewStreamResponseFactory(publisher)
	}
}
