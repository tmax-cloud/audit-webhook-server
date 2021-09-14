package dataFactory

import (
	"context"
	"os"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
	"k8s.io/klog/v2"
)

var (
	connStr        string
	ctx            context.Context
	Dbpool         *pgxpool.Pool
	DBPassWordPath string
)

func CreateConnection() {
	var err error
	// content, err := ioutil.ReadFile(DBPassWordPath)
	// if err != nil {
	// 	klog.Errorln(err)
	// 	return
	// }
	// dbRootPW := string(content)
	dbRootPW := os.Getenv("timescaledb_password")

	connStr = "postgres://postgres:{DB_ROOT_PW}@timescaledb-service.hypercloud5-system.svc.cluster.local:5432/postgres"
	// 치환
	connStr = strings.Replace(connStr, "{DB_ROOT_PW}", dbRootPW, -1)
	ctx = context.Background()

	Dbpool, err = pgxpool.Connect(ctx, connStr)
	if err != nil {
		klog.Errorf("Unable to connect to database: %v\n", err)
		panic(err)
	}
	klog.Infoln("DB Connection Success!!")
}
