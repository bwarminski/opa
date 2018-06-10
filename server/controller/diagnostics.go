package controller

import (
	"time"
	"context"
	"github.com/open-policy-agent/opa/metrics"
	"github.com/open-policy-agent/opa/topdown"
)

type DiagnosticsInput = func() (map[string]interface{}, error)

type diagnosticsLogger struct {
	logger     func(context.Context, *Info)
	revision   string
	explain    bool
	instrument bool
	buffer     Buffer
}

func (l diagnosticsLogger) Explain() bool {
	return l.explain
}

func (l diagnosticsLogger) Instrument() bool {
	return l.instrument
}

func (l diagnosticsLogger) Log(ctx context.Context, decisionID, remoteAddr, query string, input interface{}, results *interface{}, err error, m metrics.Metrics, tracer *topdown.BufferTracer) {

	info := &Info{
		Revision:   l.revision,
		Timestamp:  time.Now().UTC(),
		DecisionID: decisionID,
		RemoteAddr: remoteAddr,
		Query:      query,
		Input:      input,
		Results:    results,
		Error:      err,
		Metrics:    m,
	}

	if tracer != nil {
		info.Trace = *tracer
	}

	if l.logger != nil {
		l.logger(ctx, info)
	}

	if l.buffer == nil {
		return
	}

	l.buffer.Push(info)
}

func generateDiagnosticsLogger(ctx context.Context, c *Controller) (logger diagnosticsLogger) {
	// XXX(tsandall): set the decision logger on the diagnostic logger. The
	// diagnostic logger is called in all the necessary locations. The diagnostic
	// logger will make sure to call the decision logger regardless of whether a
	// diagnostic policy is configured. In the future, we can refactor this.
	defer func() {
		logger.revision = c.revision
		logger.logger = c.logger
	}()

	if c.diagnostics == nil {
		return diagnosticsLogger{}
	}

	input, err := makeDiagnosticsInput(r)
	if err != nil {
		return diagnosticsLogger{}
	}

	compiler := s.getCompiler()

	rego := rego.New(
		rego.Store(s.store),
		rego.Compiler(compiler),
		rego.Query(`data.system.diagnostics.config`),
		rego.Input(input),
	)

	output, err := rego.Eval(r.Context())
	if err != nil {
		return diagnosticsLogger{}
	}

	if len(output) == 1 {
		if config, ok := output[0].Expressions[0].Value.(map[string]interface{}); ok {
			switch config["mode"] {
			case "on":
				return diagnosticsLogger{
					buffer: s.diagnostics,
				}
			case "all":
				return diagnosticsLogger{
					buffer:     s.diagnostics,
					instrument: true,
					explain:    true,
				}
			}
		}
	}

	return diagnosticsLogger{}
}