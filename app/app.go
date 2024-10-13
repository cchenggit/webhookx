package app

import (
	"errors"
	"sync"

	"github.com/webhookx-io/webhookx/admin"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/pkg/cache"
	"github.com/webhookx-io/webhookx/pkg/log"
	"github.com/webhookx-io/webhookx/pkg/middlewares"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/proxy"
	"github.com/webhookx-io/webhookx/worker"
	"go.uber.org/zap"
)

var (
	ErrApplicationStarted = errors.New("already started")
	ErrApplicationStopped = errors.New("already stopped")
)

type Application struct {
	mux     sync.Mutex
	started bool

	stop chan struct{}

	cfg *config.Config

	log        *zap.SugaredLogger
	db         *db.DB
	queue      queue.TaskQueue
	dispatcher *dispatcher.Dispatcher
	cache      cache.Cache

	admin                *admin.Admin
	gateway              *proxy.Gateway
	worker               *worker.Worker
	observabilityManager *middlewares.ObservabilityManager
}

func NewApplication(cfg *config.Config) (*Application, error) {
	app := &Application{
		stop: make(chan struct{}),
		cfg:  cfg,
	}

	err := app.initialize()
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (app *Application) initialize() error {
	cfg := app.cfg

	log, err := log.NewZapLogger(&cfg.Log)
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(log)
	app.log = zap.S()

	// db
	db, err := db.NewDB(&cfg.DatabaseConfig)
	if err != nil {
		return err
	}
	app.db = db

	client, err := cfg.RedisConfig.GetClient()
	if err != nil {
		return err
	}
	// queue
	queue := queue.NewRedisQueue(client)
	app.queue = queue

	// cache
	app.cache = cache.NewRedisCache(client)

	app.dispatcher = dispatcher.NewDispatcher(log.Sugar(), queue, db)

	observabilityManager, err := middlewares.NewObservabilityManager(&cfg.Tracing)
	if err != nil {
		return err
	}
	app.observabilityManager = observabilityManager

	// worker
	if cfg.WorkerConfig.Enabled {
		tracer := app.observabilityManager.Tracer()
		app.worker = worker.NewWorker(&cfg.WorkerConfig, db, queue, tracer)
	}

	// admin
	if cfg.AdminConfig.IsEnabled() {
		handler := api.NewAPI(cfg, db, app.dispatcher).Handler()
		app.admin = admin.NewAdmin(cfg.AdminConfig, handler, app.observabilityManager)
	}

	// gateway
	if cfg.ProxyConfig.IsEnabled() {
		app.gateway = proxy.NewGateway(&cfg.ProxyConfig, db, app.dispatcher, app.observabilityManager)
	}

	return nil
}

func (app *Application) DB() *db.DB {
	return app.db
}

// Start starts application
func (app *Application) Start() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	if app.started {
		return ErrApplicationStarted
	}

	if app.admin != nil {
		app.admin.Start()
	}
	if app.gateway != nil {
		app.gateway.Start()
	}
	if app.worker != nil {
		app.worker.Start()
	}

	app.started = true

	return nil
}

func (app *Application) Wait() {
	<-app.stop
}

// Stop sotps application
func (app *Application) Stop() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	if !app.started {
		return ErrApplicationStopped
	}

	app.log.Infof("shutting down")

	defer func() {
		app.log.Infof("stopped")
	}()

	// TODO: timeout
	if app.admin != nil {
		app.admin.Stop()
	}
	if app.gateway != nil {
		app.gateway.Stop()
	}
	if app.worker != nil {
		app.worker.Stop()
	}

	if app.observabilityManager != nil {
		err := app.observabilityManager.Close()
		if err != nil {
			app.log.Infof("failed to call observability close: %v", err)
		}
	}

	app.started = false
	app.stop <- struct{}{}

	return nil
}
