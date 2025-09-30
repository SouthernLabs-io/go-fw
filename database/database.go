package database

import (
	"fmt"
	"strings"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/context"
)

var (
	DBCtxKey   = context.CtxKey("_fw_db")
	DBTxCtxKey = context.CtxKey("_fw_db_tx")
)

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
