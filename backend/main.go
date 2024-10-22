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
type TweetResForHTTPGet struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Date         string `json:"date"`
	Liked        int    `json:"liked"`
	Content      string `json:"content"`
	Retweet      int    `json:"retweet"`
	Figid        string `json:"figid"`
	Code         string `json:"code"`
	Errormessage string `json:"errormessage"`
	Lang         string `json:"lang"`
	Replyto      string `json:"replyto"`
	Replynumber  string `json:"replynumber"`
	Retweetto    string `json:"retweetto"`
}
type Like struct {
	TweetID string `json:"tweet_id"`
	UserID  string `json:"user_id"`
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

// 変更後
func getTweet(w http.ResponseWriter, r *http.Request) {
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
		rows, err := db.Query("SELECT id, name, date, liked, content, retweet, figid, code, errormessage, lang, replyto, replynumber, retweetto FROM tweet")
		if err != nil {
			print("search_error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//Jsonファイルにして送るプロセス
		var items []TweetResForHTTPGet
		for rows.Next() {
			var u TweetResForHTTPGet
			if err := rows.Scan(&u.Id, &u.Name, &u.Date, &u.Liked, &u.Content, &u.Retweet, &u.Figid, &u.Code, &u.Errormessage, &u.Lang, &u.Replyto, &u.Replynumber, &u.Retweetto); err != nil {
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
		var reqBody TweetResForHTTPGet
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
		current_time := t.Format("2006-01-02 15:04:05")
		//fmt.Println(id.String(), reqBody.Name, current_time, reqBody.Good, reqBody.Content, reqBody.Retweet)
		_, err2 := db.Exec("INSERT INTO tweet (id, name, date, liked, content, retweet, figid, code, errormessage, lang, replyto, replynumber, retweetto) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)", id.String(), reqBody.Name, current_time, reqBody.Liked, reqBody.Content, reqBody.Retweet, reqBody.Figid, reqBody.Code, reqBody.Errormessage, reqBody.Lang, reqBody.Replyto, reqBody.Retweetto)
		if err2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		_, err3 := db.Exec("UPDATE tweet t SET replynumber = (SELECT COUNT(*) FROM tweet t2 WHERE t2.replyto = t.tweet)")
		if err3 != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_, err4 := db.Exec("UPDATE tweet t SET retweet = (SELECT COUNT(*) FROM tweet t2 WHERE t2.retweetto = t.tweet)")
		if err4 != nil {
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
func toggleLike(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	var like Like
	if err := json.Unmarshal(body, &like); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	defer r.Body.Close()

	//var like Like
	//if err := json.NewDecoder(r.Body).Decode(&like); err != nil {
	//	http.Error(w, err.Error(), http.StatusBadRequest)
	//	return
	//}
	log.Printf("like.tweet", like.TweetID, "like.userid", like.UserID)
	res, err := db.Exec("DELETE FROM likes WHERE tweet_id = ? AND user_id = ?", like.TweetID, like.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rowsAffected, err := res.RowsAffected(); rowsAffected == 0 {
		_, err = db.Exec("INSERT INTO likes (tweet_id, user_id) VALUES (?, ?)", like.TweetID, like.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	_, err = db.Exec("UPDATE tweet SET liked = (SELECT COUNT(*) FROM likes WHERE tweet_id = ?) WHERE id = ?", like.TweetID, like.TweetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func main() {
	// ② /userでリクエストされたらnameパラメーターと一致する名前を持つレコードをJSON形式で返す
	http.HandleFunc("/tweet", getTweet)
	http.HandleFunc("/like", toggleLike)

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
