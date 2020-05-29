/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package postgres

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/constants"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain"
	commLog "github.com/intel-secl/intel-secl/v3/pkg/lib/common/log"
	commLogMsg "github.com/intel-secl/intel-secl/v3/pkg/lib/common/log/message"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"io/ioutil"
	"strings"
	"time"

	// Import driver for GORM
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var defaultLog = commLog.GetDefaultLogger()
var secLog = commLog.GetSecurityLogger()

// Config holds the configuration used for instantiating a new DataStore.
type Config struct {
	Host, Port, Dbname, User, Password, SslMode, SslCert string
	ConnRetryAttempts, ConnRetryTime int
}

type DataStore struct {
	Db  *gorm.DB
	cfg Config
}
// New returns a DataStore instance with the gorm.DB set with the postgres
func New(cfg Config) (DataStore, error) {
	defaultLog.Trace("postgres/postgres:New() Entering")
	defer defaultLog.Trace("postgres/postgres:New() Leaving")

	var store DataStore
	
	if cfg.Host == "" || cfg.Port == "" || cfg.User == "" ||
		cfg.Password == "" || cfg.Dbname == "" {
		err := errors.Errorf("postgres/postgres:New() All fields must be set (%s)", spew.Sdump(cfg))
		defaultLog.Error(err)
		secLog.Warningf("%s: Failed to connect to db, missing configuration - %s", commLogMsg.BadConnection, err)
		return store, err
	}
	
	cfg.SslMode = strings.TrimSpace(strings.ToLower(cfg.SslMode))
	if cfg.SslMode != constants.SslModeAllow && cfg.SslMode != constants.SslModePrefer &&
		cfg.SslMode != constants.SslModeVerifyCa && cfg.SslMode != constants.SslModeRequire {
		cfg.SslMode = constants.SslModeVerifyFull
	}

	var sslCertParams string
	if cfg.SslMode == "verify-ca" || cfg.SslMode == "verify-full" {
		sslCertParams = " sslrootcert=" + cfg.SslCert
	}

	store.cfg = cfg
	var db *gorm.DB
	var dbErr error
	numAttempts := cfg.ConnRetryAttempts
	for i := 0; i < numAttempts; i = i + 1 {
		retryTime := time.Duration(cfg.ConnRetryTime)
		db, dbErr = gorm.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s cfg.SslMode=%s%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Dbname, cfg.Password, cfg.SslMode, sslCertParams))
		if dbErr != nil {
			defaultLog.WithError(dbErr).Infof("postgres/postgres:New() Failed to connect to DB, retrying attempt %d/%d", i, numAttempts)
		} else {
			break
		}
		time.Sleep(retryTime * time.Second)
	}
	if dbErr != nil {
		defaultLog.WithError(dbErr).Infof("postgres/postgres:New() Failed to connect to db after %d attempts\n", numAttempts)
		secLog.Warningf("%s: Failed to connect to db after %d attempts", commLogMsg.BadConnection, numAttempts)
		return store, errors.Wrapf(dbErr, "Failed to connect to db after %d attempts", numAttempts)
	}
	store.Db = db
	return store, nil
}

func (ds *DataStore) ExecuteSql(sql *string) error {
	defaultLog.Trace("postgres/postgres:ExecuteSql() Entering")
	defer defaultLog.Trace("postgres/postgres:ExecuteSql() Leaving")

	defaultLog.Debugf("ExecuteSql: %s", *sql)
	err := ds.Db.Exec(*sql).Error
	if err != nil {
		return errors.Wrap(err, "pgdb: failed to execute sql")
	}
	return nil
}

func (ds *DataStore) ExecuteSqlFile(file string) error {
	defaultLog.Trace("postgres/postgres:ExecuteSqlFile() Entering")
	defer defaultLog.Trace("postgres/postgres:ExecuteSqlFile() Leaving")

	defaultLog.Debugf("ExecuteSqlFile: %s", file)
	c, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.Wrapf(err, "could not read sql file - %s", file)
	}
	sql := string(c)
	if err := ds.ExecuteSql(&sql); err != nil {
		return errors.Wrapf(err, "could not execute contents of sql file %s", file)
	}
	return nil
}
func (ds *DataStore) Migrate() error {
	defaultLog.Trace("postgres/postgres:Migrate() Entering")
	defer defaultLog.Trace("postgres/postgres:Migrate() Leaving")

	ds.Db.AutoMigrate(domain.FlavorGroup{})
	return nil
}

func (ds *DataStore) Close() {
	defaultLog.Trace("postgres/postgres:Close() Entering")
	defer defaultLog.Trace("postgres/postgres:Close() Leaving")

	if ds.Db != nil {
		ds.Db.Close()
	}
}