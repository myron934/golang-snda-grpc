package dbutil

import (
	"database/sql"
	"errors"
	"fmt"
	"local/sndaRpc/logHelper"
	"os"
	"reflect"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-sql-driver/mysql"
	"github.com/go-stack/stack"
)

var defaultMySQLManager *MySQLManager

func init() {
	defaultMySQLManager = NewMYSQLManager()
}

// NewMYSQLManager 返回新的mysql实例
func NewMYSQLManager() *MySQLManager {
	mng := MySQLManager{
		logger: log.NewLogfmtLogger(os.Stderr),
		dbs:    make(map[string]*sql.DB),
	}
	return &mng
}

// MySQLManager mysql管理器
type MySQLManager struct {
	logger log.Logger
	dbs    map[string]*sql.DB
}

// DefaultMySQLManager 返回默认的mysqlManager实例
func DefaultMySQLManager() *MySQLManager {
	return defaultMySQLManager
}

//Register 注册一个mysql数据库
//@param
//dbName: 数据库别名, 以后用这个表明来查找对应数据库
//dataSourceName: 数据库信息,格式为 [用户名]:[密码]@[协议]([ip:port])/[数据库名称及一些参数]. 如userplatform:userplatform@tcp(127.0.0.1:3306)/userplatform_global?charset=utf8
//maxIdleConns: 数据库连接池参数, 最大空闲连接数
//maxOpenConns: 最大允许打开的连接数
//@return: error
func (me *MySQLManager) Register(dbName, dataSourceName string, maxIdleConns, maxOpenConns int) error {
	if _, ok := me.dbs[dbName]; ok {
		return fmt.Errorf("redis named %s has exist", dbName)
	}
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return err
	}
	if maxIdleConns >= 0 {
		db.SetMaxIdleConns(maxIdleConns)
	}
	if maxOpenConns >= 0 {
		db.SetMaxOpenConns(maxOpenConns)
	}
	me.dbs[dbName] = db
	return nil
}

//Query 查询数据库,返回结果集
//@param
//dbName: 数据库别名, 以后用这个表明来查找对应数据库
//sql: sql查询语句. 如 SELECT * FROM t_test WHERE id=?
//args: sql中需要填充的参数,跟问号的数量一致
//@return []map[string]interface{}, error
func (me *MySQLManager) Query(dbName, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	level.Info(me.logger).Log("ts", time.Now().Format("2006-01-02 15:04:05.000.000000"), "caller", stack.Caller(1), "sql", sql, "params", fmt.Sprint(args))
	db, ok := me.dbs[dbName]
	if !ok {
		return nil, fmt.Errorf("Can't find DB named %s ", dbName)
	}
	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	colLen := len(colTypes)

	//表
	data := make([]map[string]interface{}, 0)
	for rows.Next() {
		//行table(key,value)
		rowData := make(map[string]interface{})
		//行数据(value)
		valueList := make([]interface{}, 0, colLen)
		for _, ctp := range colTypes {
			dbType := ctp.DatabaseTypeName()
			switch dbType {
			case "INT", "DECIMAL", "BIGINT":
				valueList = append(valueList, new(int))
			default:
				valueList = append(valueList, new(string))
			}
			//valueList=append(valueList,reflect.New(ctp.ScanType()).Interface())
		}
		if err := rows.Scan(valueList...); err != nil {
			level.Error(me.logger).Log("err", err, "caller", stack.Caller(1))
			continue
		}
		for i, value := range valueList {
			rowData[colTypes[i].Name()] = reflect.ValueOf(value).Elem().Interface()
		}
		data = append(data, rowData)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

//Exec 执行单句增删改语句(UPDATE, DELETE, INSERT),返回执行结果
//@param
//dbName: 数据库别名, 以后用这个表明来查找对应数据库
//sql: sql查询语句. 如 SELECT * FROM t_test WHERE id=?
//args: sql中需要填充的参数,跟问号的数量一致
//@return sql.Result, error
func (me *MySQLManager) Exec(dbName string, sql string, args ...interface{}) (sql.Result, error) {
	db, ok := me.dbs[dbName]
	if !ok {
		return nil, fmt.Errorf("Can't find DB named %s ", dbName)
	}
	return db.Exec(sql, args...)
}

//Begin 在多语句执行,需要用到事务保证一致性时,该方法返回原始的事务,给用户自行处理事务
//@param
//dbName: 数据库别名, 以后用这个表明来查找对应数据库
//@return *sql.Tx, error
func (me *MySQLManager) Begin(dbName string) (*sql.Tx, error) {
	db, ok := me.dbs[dbName]
	if !ok {
		return nil, fmt.Errorf("Can't find DB named %s ", dbName)
	}
	return db.Begin()
}

//Prepare 返回原生statement
//@param
//dbName: 数据库别名, 以后用这个表明来查找对应数据库
//@return *sql.Stmt, error
func (me *MySQLManager) Prepare(dbName string, sql string) (*sql.Stmt, error) {
	db, ok := me.dbs[dbName]
	if !ok {
		return nil, fmt.Errorf("Can't find DB named %s ", dbName)
	}
	return db.Prepare(sql)
}

//DB 返回Db实例
//@param
//dbName: 数据库别名, 以后用这个表明来查找对应数据库
//@return *sql.DB, error
func (me *MySQLManager) DB(dbName string) (db *sql.DB, ok bool) {
	db, ok = me.dbs[dbName]
	return
}

//SetLogger 设置logger
func (me *MySQLManager) SetLogger(logger log.Logger) error {
	if logger == nil {
		return errors.New("nil logger not allowed")
	}
	me.logger = logger
	mysql.SetLogger(logHelper.MysqlLogger(logger))
	return nil
}
