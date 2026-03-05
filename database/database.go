package database

import (
	"fmt"
	"strings"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/context"
)

type DBCtxCustomKeyType struct{}

var (
	DBCtxKey   = context.CtxKey("_fw_db")
	DBTxCtxKey = context.CtxKey("_fw_db_tx")

	// This must be non string so it forces the creation of a new context when a KeyValueContext is used so we can specify what is the current custom key to use by stacking contexts.
	DBCtxCustomKey = DBCtxCustomKeyType{}
)

func CreateDBCtxKey(name string) context.CtxKey {
	return context.CtxKey(fmt.Sprintf("%s:%s", DBCtxKey, name))
}

func CreateDBTxCtxKey(name string) context.CtxKey {
	return context.CtxKey(fmt.Sprintf("%s:%s", DBTxCtxKey, name))
}

const (
	ErrCodeCommitFailed   = "DB_COMMIT_FAILED"
	ErrCodeRollbackFailed = "DB_ROLLBACK_FAILED"
)

func CreateDBName(conf config.Config) string {
	return strings.ReplaceAll(
		strings.ToLower(fmt.Sprintf("%s_%s", conf.Name, conf.Env.Name)),
		"-",
		"_",
	)
}
