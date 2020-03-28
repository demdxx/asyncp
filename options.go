package asyncp

// Options of the mux server
type Options struct {
	PanicHandler          func(Task, Event, interface{})
	ErrorHandler          func(Task, Event, error)
	StreamResponseFactory ResponseWriterFactory
}

// Option of the task configuration
type Option func(opt *Options)

// WithPanicHandler puts panic handler to the Mux option
func WithPanicHandler(h func(Task, Event, interface{})) Option {
	return func(opt *Options) {
		opt.PanicHandler = h
	}
}

// WithErrorHandler puts error handler to the Mux option
func WithErrorHandler(h func(Task, Event, error)) Option {
	return func(opt *Options) {
		opt.ErrorHandler = h
	}
}

// WithStreamResponseFactory set option with stream factory
func WithStreamResponseFactory(publisher Publisher) Option {
	return func(opt *Options) {
		opt.StreamResponseFactory = NewStreamResponseFactory(publisher)
	}
}
