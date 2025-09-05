package datasource

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"obsessiontech/common/config"
	"obsessiontech/common/context"
)

var Config struct {
	User               string
	Password           string
	URL                string
	MaxIdle            int
	MaxConn            int
	MaxConnLifeTimeMin time.Duration
}

var conn *sql.DB

func GetConn() *sql.DB {
	if conn == nil {
		log.Fatalln("ds nil")
	}
	return conn
}

func init() {
	config.GetConfig("config.yaml", &Config)

	dbURL := fmt.Sprintf("%s:%s@%s", Config.User, Config.Password, Config.URL)

	fmt.Println(dbURL)

	db, err := sql.Open("mysql", dbURL)

	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	if db != nil {
		if Config.MaxConnLifeTimeMin > 0 {
			db.SetConnMaxLifetime(Config.MaxConnLifeTimeMin * time.Minute)
		}
		if Config.MaxConn > 0 {
			db.SetMaxOpenConns(Config.MaxConn)
		}
		if Config.MaxIdle > 0 {
			db.SetMaxIdleConns(Config.MaxIdle)
		}
		conn = db
	}
}

func Txn(txnFunc func(*sql.Tx), opts ...*sql.TxOptions) (e error) {

	ctx, cancel := context.GetContext()
	defer cancel()

	var opt *sql.TxOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	txn, err := GetConn().BeginTx(ctx, opt)
	if err != nil {
		return err
	}

	defer func() {
		if err := recover(); err != nil {
			if e1 := txn.Rollback(); e1 != nil {
				log.Println("error txn rollback: ", e1)
			}

			if e1, ok := err.(error); ok {
				e = e1
			} else {
				log.Println("recover: ", err)
				e = errors.New("事务失败")
			}
		} else {
			txn.Commit()
		}
	}()

	txnFunc(txn)

	return nil
}
