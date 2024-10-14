package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/oklog/ulid"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//	type UserResForHTTPGet struct {
//		Id   string `json:"id"`
//		Name string `json:"name"`
//		Age  int    `json:"age"`
//	}
type UserResForHTTPGet struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Date    string `json:"date"`
	Good    int    `json:"good"`
	Content string `json:"content"`
	Retweet int    `json:"retweet"`
}

type responseMessage struct {
	Message string `json:"message"`
}

// ① GoプログラムからMySQLへ接続
var db *sql.DB

func init() {
	// ①-1
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlUserPwd := os.Getenv("MYSQL_PASSWORD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	// ①-2
	_db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@%s/%s", mysqlUser, mysqlUserPwd, mysqlHost, mysqlDatabase))
	if err != nil {
		log.Fatalf("fail: sql.Open, %v\n", err)
	}
	// ①-3
	if err := _db.Ping(); err != nil {
		log.Fatalf("fail: _db.Ping, %v\n", err)
	}
	db = _db
}

// ② /userでリクエストされたらnameパラメーターと一致する名前を持つレコードをJSON形式で返す

//差分
//func handler(w http.ResponseWriter, r *http.Request) {
//	switch r.Method {
//	case http.MethodGet:
//		// ②-1
//		name := r.URL.Query().Get("name") // To be filled
//		if name == "" {
//			log.Println("fail: name is empty")
//			w.WriteHeader(http.StatusBadRequest)
//			return
//		}
//
//		// ②-2
//		rows, err := db.Query("SELECT id, name, age FROM user WHERE name = ?", name)
//		if err != nil {
//			log.Printf("fail: db.Query, %v\n", err)
//			w.WriteHeader(http.StatusInternalServerError)
//			return
//		}
//
//		// ②-3
//		users := make([]UserResForHTTPGet, 0)
//		for rows.Next() {
//			var u UserResForHTTPGet
//			if err := rows.Scan(&u.Id, &u.Name, &u.Age); err != nil {
//				log.Printf("fail: rows.Scan, %v\n", err)
//
//				if err := rows.Close(); err != nil { // 500を返して終了するが、その前にrowsのClose処理が必要
//					log.Printf("fail: rows.Close(), %v\n", err)
//				}
//				w.WriteHeader(http.StatusInternalServerError)
//				return
//			}
//			users = append(users, u)
//		}
//
//		// ②-4
//		bytes, err := json.Marshal(users)
//		if err != nil {
//			log.Printf("fail: json.Marshal, %v\n", err)
//			w.WriteHeader(http.StatusInternalServerError)
//			return
//		}
//		w.Header().Set("Content-Type", "application/json")
//		w.Write(bytes)
//	case http.MethodPost:
//		//postしたファイルを読み取るプロセス
//		body, err := io.ReadAll(r.Body)
//		if err != nil {
//			w.WriteHeader(http.StatusInternalServerError)
//		}
//		var reqBody UserResForHTTPGet
//		if err := json.Unmarshal(body, &reqBody); err != nil {
//			w.WriteHeader(http.StatusInternalServerError)
//		}
//		defer r.Body.Close()
//
//		if len(reqBody.Name) > 50 || reqBody.Name == "" || reqBody.Age < 20 || reqBody.Age > 80 {
//			w.WriteHeader(http.StatusBadRequest)
//		}
//
//		//ULIDを用いてidを生成するプロセス
//		t := time.Now()
//		entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
//		id := ulid.MustNew(ulid.Timestamp(t), entropy)
//
//		//データベースに書き込む
//		_, err2 := db.Exec("INSERT INTO user (id, name, age) VALUES (?, ?, ?)", id.String(), reqBody.Name, reqBody.Age)
//		if err2 != nil {
//			w.WriteHeader(http.StatusInternalServerError)
//		}
//
//		//書き込みができたらステータスを変更し、idを出力
//		w.WriteHeader(http.StatusOK)
//		bytes, err3 := json.Marshal(responseMessage{
//			Message: "id :" + id.String(),
//		})
//		if err3 != nil {
//			w.WriteHeader(http.StatusInternalServerError)
//			return
//		}
//		w.Header().Set("Content-Type", "application/json")
//		w.Write(bytes)
//		return
//	default:
//		log.Printf("fail: HTTP Method is %s\n", r.Method)
//		w.WriteHeader(http.StatusBadRequest)
//		return
//	}
//}

// 変更後
func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodGet:
		//Getクエリが来たらデータベースを検索
		rows, err := db.Query("SELECT id, name, date, good, content, retweet FROM tweet")
		if err != nil {
			print("search_error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//Jsonファイルにして送るプロセス
		var items []UserResForHTTPGet
		for rows.Next() {
			var u UserResForHTTPGet
			if err := rows.Scan(&u.Id, &u.Name, &u.Date, &u.Good, &u.Content, &u.Retweet); err != nil {
				print("error")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			items = append(items, u)
		}
		//Json形式にして送る
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(items); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

	case http.MethodPost:
		//postしたファイルを読み取るプロセス
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		var reqBody UserResForHTTPGet
		if err := json.Unmarshal(body, &reqBody); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		defer r.Body.Close()

		//if len(reqBody.Name) > 50 || reqBody.Name == "" || reqBody.Age < 20 || reqBody.Age > 80 {
		//	w.WriteHeader(http.StatusBadRequest)
		//}

		//ULIDを用いてidを生成するプロセス
		t := time.Now()
		entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
		id := ulid.MustNew(ulid.Timestamp(t), entropy)

		//データベースに書き込む
		_, err2 := db.Exec("INSERT INTO tweet (id, name, date, good, content, retweet) VALUES (?, ?, ?, ?, ?, ?)", id.String(), reqBody.Name, t.String(), reqBody.Good, reqBody.Content, reqBody.Retweet)
		if err2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		//書き込みができたらステータスを変更し、idを出力
		w.WriteHeader(http.StatusOK)
		bytes, err3 := json.Marshal(responseMessage{
			Message: "id :" + id.String(),
		})
		if err3 != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
		return
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func main() {
	// ② /userでリクエストされたらnameパラメーターと一致する名前を持つレコードをJSON形式で返す
	http.HandleFunc("/user", handler)

	// ③ Ctrl+CでHTTPサーバー停止時にDBをクローズする
	closeDBWithSysCall()

	// 8000番ポートでリクエストを待ち受ける
	log.Println("Listening...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// ③ Ctrl+CでHTTPサーバー停止時にDBをクローズする
func closeDBWithSysCall() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sig
		log.Printf("received syscall, %v", s)

		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
		log.Printf("success: db.Close()")
		os.Exit(0)
	}()
} //TIP See GoLand help at <a href="https://www.jetbrains.com/help/go/">jetbrains.com/help/go/</a>.
// Also, you can try interactive lessons for GoLand by selecting 'Help | Learn IDE Features' from the main menu.
