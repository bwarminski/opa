// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package controller

import (
	"sync"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/plugins"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/watch"
	"context"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/topdown"
	"github.com/open-policy-agent/opa/metrics"
)

type Controller struct {
	mtx               sync.RWMutex
	partials          map[string]rego.PartialResult
	store             storage.Store
	manager           *plugins.Manager
	watcher           *watch.Watcher
	decisionIDFactory func() string
	diagnostics       Buffer
	revision          string
	errLimit          int
	logger            func(context.Context, *Info)
}

func New() *Controller {
	c := Controller{}
	return &c
}

func (c *Controller) WithStore(store storage.Store) *Controller {
	c.store = store
	return c
}

func (c *Controller) WithManager(manager *plugins.Manager) *Controller {
	c.manager = manager
	return c
}

func (c *Controller) WithCompilerErrorLimit(limit int) *Controller {
	c.errLimit = limit
	return c
}

func (c *Controller) WithDiagnosticsBuffer(buf Buffer) *Controller {
	c.diagnostics = buf
	return c
}

func (c *Controller) WithDecisionLogger(logger func(context.Context, *Info)) *Controller {
	c.logger = logger
	return c
}

func (c *Controller) WithDecisionIDFactory(f func() string) *Controller {
	c.decisionIDFactory = f
	return c
}

func (c *Controller) Init(ctx context.Context) error {
	txn, err := c.store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		return err
	}

	// Register triggers so that if runtime reloads the policies, the
	// server sees the change.
	config := storage.TriggerConfig{
		OnCommit: c.reload,
	}

	if _, err := c.store.Register(ctx, txn, config); err != nil {
		c.store.Abort(ctx, txn)
		return err
	}

	c.manager.RegisterCompilerTrigger(c.migrateWatcher)

	c.watcher, err = watch.New(ctx, c.store, c.GetCompiler(), txn)
	if err != nil {
		return err
	}

	c.partials = map[string]rego.PartialResult{}

	return c.store.Commit(ctx, txn)
}

func (c *Controller) NewTransaction(ctx context.Context, params ...storage.TransactionParams) (storage.Transaction, error) {
	return c.store.NewTransaction(ctx, params...)
}

func (c *Controller) Abort(ctx context.Context, txn storage.Transaction) {
	c.store.Abort(ctx, txn)
}

func (c *Controller) reload(ctx context.Context, txn storage.Transaction, event storage.TriggerEvent) {

	value, err := c.store.Read(ctx, txn, storage.MustParsePath("/system/bundle/manifest/revision"))
	if err == nil {
		revision, ok := value.(string)
		if !ok {
			panic("bad revision value")
		}
		c.revision = revision
	} else if err != nil {
		if !storage.IsNotFound(err) {
			panic(err)
		}
	}

	c.partials = map[string]rego.PartialResult{}
}

func (c *Controller) migrateWatcher(txn storage.Transaction) {
	var err error
	c.watcher, err = c.watcher.Migrate(c.manager.GetCompiler(), txn)
	if err != nil {
		// The only way migration can fail is if the old watcher is closed or if
		// the new one cannot register a trigger with the store. Since we're
		// using an inmem store with a write transaction, neither of these should
		// be possible.
		panic(err)
	}
}

// Should probably keep store and compiler private
func (c *Controller) GetCompiler() *ast.Compiler {
	return c.manager.GetCompiler()
}

func (c *Controller) GetStore() storage.Store {
	return c.store
}

func (c *Controller) GetDiagnostics() Buffer {
	return c.diagnostics
}

type QueryRequest struct {
	Query       string
	ParsedInput ast.Value
	Instrument  bool
	Tracer      *topdown.BufferTracer
	Transaction storage.Transaction
	RawInput    interface{}
}

type QueryResponse struct {
	Results rego.ResultSet
	Metrics metrics.Metrics
}

func (c *Controller) ExecQuery(ctx context.Context, request QueryRequest) (response QueryResponse, err error) {
	response.Metrics = metrics.New()
	compiler := c.GetCompiler()
	rego := rego.New(
		rego.Store(c.store),
		rego.Compiler(compiler),
		rego.Query(request.Query),
		rego.ParsedInput(request.ParsedInput),
		rego.Metrics(response.Metrics),
		rego.Instrument(request.Instrument),
		rego.Tracer(request.Tracer),
		rego.Input(request.RawInput),
		rego.Transaction(request.Transaction),
	)

	response.Results, err = rego.Eval(ctx)
	return
}

func (c *Controller) TxnExecQuery(ctx context.Context, txn storage.Transaction, request QueryRequest) (response QueryResponse, err error) {
	response.Metrics = metrics.New()
	compiler := c.GetCompiler()
	rego := rego.New(
		rego.Store(c.store),
		rego.Compiler(compiler),
		rego.Query(request.Query),
		rego.ParsedInput(request.ParsedInput),
		rego.Metrics(response.Metrics),
		rego.Instrument(request.Instrument),
		rego.Tracer(request.Tracer),
	)

	response.Results, err = rego.Eval(ctx)
	return
}



