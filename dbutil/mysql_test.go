package dbutil

import (
	"database/sql"
	"fmt"
	"local/sndaRpc/util"
	"testing"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

func TestMysql1(t *testing.T) {
	db, err := sql.Open("mysql", "userplatform:userplatform@tcp(127.0.0.1:3306)/userplatform_global?charset=utf8")
	if err != nil {
		t.Error(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
		return
	}
	defer db.Close()

	// Prepare statement for inserting data
	stmtIns, err := db.Prepare("INSERT INTO circle_first_ad_more(ad_id, circle_id) VALUES(?,?)") // ? = placeholder
	if err != nil {
		t.Error(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates
	// Insert square numbers for 0-24 in the database
	for i := 0; i < 5; i++ {
		_, err := stmtIns.Exec(i, (i * i)) // Insert tuples (i, i^2)
		if err != nil {
			t.Errorf("err: %v", err) // Just for example purpose. You should use proper error handling instead of panic
		}
	}

	// Prepare statement for reading data
	stmtOut, err := db.Prepare("SELECT id,ad_text FROM `circle_first_ad` WHERE circle_id=?")
	if err != nil {
		t.Error(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
		return
	}
	defer stmtOut.Close()

	var ad_id int // we "scan" the result in here
	var ad_text string
	// Query the square-number of 13
	rows, err := stmtOut.Query(42)
	if err != nil {
		t.Error(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
		return
	}
	for rows.Next() {
		if err := rows.Scan(&ad_id, &ad_text); err != nil {
			t.Error(err.Error())
		}
		fmt.Printf("ad_id: %d; ad_text: %s\n", ad_id, ad_text)
	}
	// Query another number.. 1 maybe?
	err = stmtOut.QueryRow(10039).Scan(&ad_id, &ad_text) // WHERE number = 1
	if err != nil {
		t.Error(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
		return
	}
	fmt.Printf("ad_id: %d; ad_text: %s\n", ad_id, ad_text)
}

func TestMysqlQuery(t *testing.T) {
	DefaultMySQLManager().Register("default", "userplatform:userplatform@tcp(127.0.0.1:3306)/userplatform_global?charset=utf8", 10, 10)
	fmt.Println("---")
	data, err := DefaultMySQLManager().Query(
		"default",
		"SELECT * FROM circle_first_ad_more where circle_id=?",
		0,
	)
	if err != nil {
		t.Error(err)
		return
	}
	for _, row := range data {
		id := util.Int(row["ad_id"], 0)
		text := util.String(row["rate"], "0.0")
		createTime := util.String(row["create_time"], "")
		fmt.Printf("id=%d, rate=%s, time=%v\n", id, text, createTime)
	}
}

func TestMysqlExec(t *testing.T) {
	DefaultMySQLManager().Register("default", "userplatform:userplatform@tcp(127.0.0.1:3306)/userplatform_global?charset=utf8", 10, 10)
	rst, err := DefaultMySQLManager().Exec(
		"default",
		"INSERT INTO circle_first_ad_more(ad_id,rate, circle_id) VALUES(?,?,?)",
		0, 1, 2,
	)

	if err != nil {
		if nerr, ok := err.(*mysql.MySQLError); ok {
			t.Error("errno=%d  msg=%s", nerr.Number, nerr.Message)
		} else {
			t.Error(err.Error())
		}
		return
	}
	lastId, err := rst.LastInsertId()
	rowCount, err := rst.RowsAffected()
	fmt.Printf("lastid=%d AffectedRow=%d", lastId, rowCount)
}
