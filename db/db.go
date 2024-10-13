package db

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/transaction"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type DB struct {
	DB  *sqlx.DB
	log *zap.SugaredLogger

	Workspaces       dao.WorkspaceDAO
	Endpoints        dao.EndpointDAO
	EndpointsWS      dao.EndpointDAO
	Events           dao.EventDAO
	EventsWS         dao.EventDAO
	Attempts         dao.AttemptDAO
	AttemptsWS       dao.AttemptDAO
	Sources          dao.SourceDAO
	SourcesWS        dao.SourceDAO
	AttemptDetails   dao.AttemptDetailDAO
	AttemptDetailsWS dao.AttemptDetailDAO
	Plugins          dao.PluginDAO
	PluginsWS        dao.PluginDAO
}

func initSqlxDB(cfg *config.DatabaseConfig) (*sqlx.DB, error) {
	db, err := cfg.GetDB()
	// db.SetMaxOpenConns(100)
	// db.SetMaxIdleConns(100)
	// db.SetConnMaxLifetime(time.Hour)
	// db.SetConnMaxIdleTime(time.Hour)
	if err != nil {
		return nil, err
	}
	
	return sqlx.NewDb(db, "postgres"), nil
}

func NewDB(cfg *config.DatabaseConfig) (*DB, error) {
	sqlxDB, err := initSqlxDB(cfg)
	if err != nil {
		return nil, err
	}

	db := &DB{
		DB:               sqlxDB,
		log:              zap.S(),
		Workspaces:       dao.NewWorkspaceDAO(sqlxDB),
		Endpoints:        dao.NewEndpointDAO(sqlxDB, false),
		EndpointsWS:      dao.NewEndpointDAO(sqlxDB, true),
		Events:           dao.NewEventDao(sqlxDB, false),
		EventsWS:         dao.NewEventDao(sqlxDB, true),
		Attempts:         dao.NewAttemptDao(sqlxDB, false),
		AttemptsWS:       dao.NewAttemptDao(sqlxDB, true),
		Sources:          dao.NewSourceDAO(sqlxDB, false),
		SourcesWS:        dao.NewSourceDAO(sqlxDB, true),
		AttemptDetails:   dao.NewAttemptDetailDao(sqlxDB, false),
		AttemptDetailsWS: dao.NewAttemptDetailDao(sqlxDB, true),
		Plugins:          dao.NewPluginDAO(sqlxDB, false),
		PluginsWS:        dao.NewPluginDAO(sqlxDB, true),
	}

	return db, nil
}

func (db *DB) Ping() error {
	return db.DB.Ping()
}

var tracer = otel.Tracer("db")

func (db *DB) TX(ctx context.Context, fn func(ctx context.Context) error) error {
	_, span := tracer.Start(ctx, "transaction")
	defer span.End()

	tx, err := db.DB.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if err := recover(); err != nil {
			db.log.Errorf("[db] panic recovered: %v", err)
			if rbErr := tx.Rollback(); rbErr != nil {
				db.log.Errorf("[db] failed to rollback the tx: %v", rbErr)
			}
			panic(err)
		}
	}()

	ctx = transaction.WithTx(ctx, tx)

	err = fn(ctx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Wrap(err, rbErr.Error())
		}
		return err
	}

	return tx.Commit()
}

func (db *DB) Truncate(table string) error {
	sql := fmt.Sprintf("DELETE FROM %s", table)
	_, err := db.DB.Exec(sql)
	return err
}
