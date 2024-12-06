package main

import (
	"cloud.google.com/go/vertexai/genai"
	"context"
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

type TweetResForHTTPGet struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Date           string `json:"date"`
	Liked          int    `json:"liked"`
	Content        string `json:"content"`
	Retweet        int    `json:"retweet"`
	Figid          string `json:"figid"`
	Code           string `json:"code"`
	Errormessage   string `json:"errormessage"`
	Lang           string `json:"lang"`
	Replyto        string `json:"replyto"`
	Replynumber    int    `json:"replynumber"`
	Retweetto      string `json:"retweetto"`
	Retweetcomment string `json:"retweetcomment"`
}

type FollowResForHTTPGet struct {
	Follower string `json:"follower"`
	Followed string `json:"followed"`
}
type FollowreqResForHTTPGet struct {
	Followerreq string `json:"followerreq"`
	Followedreq string `json:"followedreq"`
}
type Like struct {
	TweetID string `json:"tweet_id"`
	UserID  string `json:"user_id"`
}

type responseMessage struct {
	Message string `json:"message"`
}

type DebugRequest struct {
	Code    string            `json:"code"`
	Options map[string]string `json:"options"`
}

type User_id struct {
	User_id string `json:"user_id"`
}

type LikeResForHTTPGet struct {
	Tweet_id string `json:"tweet_id"`
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
		rows, err := db.Query("SELECT id, name, date, liked, content, retweet, figid, code, errormessage, lang, replyto, replynumber, retweetto, retweetcomment FROM tweet")
		if err != nil {
			print("search_error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//Jsonファイルにして送るプロセス
		var items []TweetResForHTTPGet
		for rows.Next() {
			var u TweetResForHTTPGet
			if err := rows.Scan(&u.Id, &u.Name, &u.Date, &u.Liked, &u.Content, &u.Retweet, &u.Figid, &u.Code, &u.Errormessage, &u.Lang, &u.Replyto, &u.Replynumber, &u.Retweetto, &u.Retweetcomment); err != nil {
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
		//fmt.Println(id.String(), reqBody.Name, current_time, reqBody.Liked, reqBody.Content, reqBody.Retweet, )
		_, err2 := db.Exec("INSERT INTO tweet (id, name, date, liked, content, retweet, figid, code, errormessage, lang, replyto, replynumber, retweetto, retweetcomment) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)", id.String(), reqBody.Name, current_time, reqBody.Liked, reqBody.Content, reqBody.Retweet, reqBody.Figid, reqBody.Code, reqBody.Errormessage, reqBody.Lang, reqBody.Replyto, reqBody.Retweetto, reqBody.Retweetcomment)
		if err2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		_, err3 := db.Exec("UPDATE tweet t JOIN(SELECT replyto, COUNT(*) AS reply_count FROM tweet GROUP BY replyto) AS counts ON t.id = counts.replyto SET t.replynumber = counts.reply_count WHERE counts.replyto IS NOT NULL")
		if err3 != nil {
			fmt.Println("err3")
			w.WriteHeader(http.StatusInternalServerError)
		}
		_, err4 := db.Exec("UPDATE tweet t JOIN(SELECT retweetto, COUNT(*) AS retweet_count FROM tweet GROUP BY retweetto) AS counts ON t.id = counts.retweetto SET t.retweet = counts.retweet_count WHERE counts.retweetto IS NOT NULL")
		if err4 != nil {
			fmt.Println("err4")
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

func getLike(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			print("read_error")
			w.WriteHeader(http.StatusInternalServerError)
		}
		var reqBody User_id
		if err := json.Unmarshal(body, &reqBody); err != nil {
			print("convert_error")
			w.WriteHeader(http.StatusInternalServerError)
		}
		defer r.Body.Close()

		//Getクエリが来たらデータベースを検索
		rows, err := db.Query("SELECT tweet_id FROM likes WHERE user_id = ?", reqBody.User_id)
		if err != nil {
			print("search_error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//Jsonファイルにして送るプロセス
		var items []LikeResForHTTPGet
		for rows.Next() {
			var u LikeResForHTTPGet
			if err := rows.Scan(&u.Tweet_id); err != nil {
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

func follow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT follower, followed FROM follow")
		if err != nil {
			print("search_error in follow")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//Jsonファイルにして送るプロセス
		var items []FollowResForHTTPGet
		for rows.Next() {
			var u FollowResForHTTPGet
			if err := rows.Scan(&u.Follower, &u.Followed); err != nil {
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
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		var reqBody FollowResForHTTPGet
		if err := json.Unmarshal(body, &reqBody); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		defer r.Body.Close()
		res, err := db.Exec("DELETE FROM followreq WHERE followerreq = ? AND followedreq = ?", reqBody.Follower, reqBody.Followed)
		if rowsAffected, err := res.RowsAffected(); rowsAffected == 0 {
			fmt.Println("follow request do not exist")
			http.Error(w, "follow request do not exist", http.StatusInternalServerError)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		_, err2 := db.Exec("INSERT INTO follow (follower, followed) VALUES (?, ?)", reqBody.Follower, reqBody.Followed)
		if err2 != nil {
			fmt.Println("error in writing into follow")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func followreq(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT followerreq, followedreq FROM followreq")
		if err != nil {
			print("search_error in followreq")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//Jsonファイルにして送るプロセス
		var items []FollowreqResForHTTPGet
		for rows.Next() {
			var u FollowreqResForHTTPGet
			if err := rows.Scan(&u.Followerreq, &u.Followedreq); err != nil {
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
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		var reqBody FollowreqResForHTTPGet
		if err := json.Unmarshal(body, &reqBody); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		defer r.Body.Close()

		res, err := db.Exec("DELETE FROM followreq WHERE followerreq = ? AND followedreq = ?", reqBody.Followerreq, reqBody.Followedreq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if rowsAffected, err := res.RowsAffected(); rowsAffected == 0 {
			_, err = db.Exec("INSERT INTO followreq (followerreq, followedreq) VALUES (?, ?)", reqBody.Followerreq, reqBody.Followedreq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		//_, err2 := db.Exec("INSERT INTO followreq (followerreq, followedreq) VALUES (?, ?)", reqBody.Followerreq, reqBody.Followedreq)
		//if err2 != nil {
		//	fmt.Println("error in writing into follow")
		//	w.WriteHeader(http.StatusInternalServerError)
		//}
	}
}

func askGemini(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	var projectId = "term6-riku-yagashi"
	var region = "us-central1"
	var modelName = "gemini-1.0-pro-vision"

	body, err := io.ReadAll(r.Body)
	if err != nil {
		print(err)
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	var reqBody TweetResForHTTPGet
	if err := json.Unmarshal(body, &reqBody); err != nil {
		print(err)
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	fmt.Println(reqBody)
	ctx := context.Background()
	client, err := genai.NewClient(ctx, projectId, region)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Errorf("error creating client: %v", err)
		return
	}
	defer client.Close()

	gemini := client.GenerativeModel(modelName)
	chat := gemini.StartChat()

	res, err := chat.SendMessage(
		ctx,
		genai.Text("Help me with debugging code"+"comment is"+reqBody.Content+". code is "+reqBody.Code+". language is"+reqBody.Lang+". Error message is "+reqBody.Errormessage))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res_code, err_code := chat.SendMessage(
		ctx,
		genai.Text("Please show me only code"))
	if err_code != nil {
		fmt.Println("cannot show code")
		http.Error(w, err_code.Error(), http.StatusInternalServerError)
		return
	}

	res_error, err_error := chat.SendMessage(
		ctx,
		genai.Text("execute the previous code and show me only result of code"))
	if err_error != nil {
		fmt.Println("cannot execute")
		http.Error(w, err_error.Error(), http.StatusInternalServerError)
		return
	}

	rb, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(rb))
	fmt.Println(res.Candidates[0].Content.Parts[0])

	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)

	//データベースに書き込む
	current_time := t.Format("2006-01-02 15:04:05")
	//fmt.Println(id.String(), reqBody.Name, current_time, reqBody.Liked, reqBody.Content, reqBody.Retweet, )
	_, err2 := db.Exec("INSERT INTO tweet (id, name, date, liked, content, retweet, figid, code, errormessage, lang, replyto, replynumber, retweetto, retweetcomment) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)", id.String(), "Gemini", current_time, reqBody.Liked, res.Candidates[0].Content.Parts[0], reqBody.Retweet, reqBody.Figid, res_code.Candidates[0].Content.Parts[0], res_error.Candidates[0].Content.Parts[0], reqBody.Lang, reqBody.Replyto, reqBody.Retweetto, reqBody.Retweetcomment)
	if err2 != nil {
		fmt.Println(err2)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err3 := db.Exec("UPDATE tweet t JOIN(SELECT replyto, COUNT(*) AS reply_count FROM tweet GROUP BY replyto) AS counts ON t.id = counts.replyto SET t.replynumber = counts.reply_count WHERE counts.replyto IS NOT NULL")
	if err3 != nil {
		fmt.Println(err3)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err4 := db.Exec("UPDATE tweet t JOIN(SELECT retweetto, COUNT(*) AS retweet_count FROM tweet GROUP BY retweetto) AS counts ON t.id = counts.retweetto SET t.retweet = counts.retweet_count WHERE counts.retweetto IS NOT NULL")
	if err4 != nil {
		fmt.Println(err4)
		w.WriteHeader(http.StatusInternalServerError)
		return
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
}

func executeOnGemini(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	var projectId = "term6-riku-yagashi"
	var region = "us-central1"
	var modelName = "gemini-1.0-pro-vision"

	body, err := io.ReadAll(r.Body)
	if err != nil {
		print(err)
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	var reqBody TweetResForHTTPGet
	if err := json.Unmarshal(body, &reqBody); err != nil {
		print(err)
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	fmt.Println(reqBody)
	ctx := context.Background()
	client, err := genai.NewClient(ctx, projectId, region)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Errorf("error creating client: %v", err)
		return
	}
	defer client.Close()

	gemini := client.GenerativeModel(modelName)
	chat := gemini.StartChat()

	res, err := chat.SendMessage(
		ctx,
		genai.Text("Execute this code on gemini."+"code is "+reqBody.Code+". language is"+reqBody.Lang+"and show me the result of it"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res_error, err_error := chat.SendMessage(
		ctx,
		genai.Text("execute the previous code and show me only result of code"))
	if err_error != nil {
		fmt.Println("cannot execute")
		http.Error(w, err_error.Error(), http.StatusInternalServerError)
		return
	}

	rb, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(rb))
	fmt.Println(res.Candidates[0].Content.Parts[0])

	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)

	//データベースに書き込む
	current_time := t.Format("2006-01-02 15:04:05")
	//fmt.Println(id.String(), reqBody.Name, current_time, reqBody.Liked, reqBody.Content, reqBody.Retweet, )
	_, err2 := db.Exec("INSERT INTO tweet (id, name, date, liked, content, retweet, figid, code, errormessage, lang, replyto, replynumber, retweetto, retweetcomment) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)", id.String(), "Gemini", current_time, reqBody.Liked, res.Candidates[0].Content.Parts[0], reqBody.Retweet, reqBody.Figid, reqBody.Code, res_error.Candidates[0].Content.Parts[0], reqBody.Lang, reqBody.Replyto, reqBody.Retweetto, reqBody.Retweetcomment)
	if err2 != nil {
		fmt.Println(err2)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err3 := db.Exec("UPDATE tweet t JOIN(SELECT replyto, COUNT(*) AS reply_count FROM tweet GROUP BY replyto) AS counts ON t.id = counts.replyto SET t.replynumber = counts.reply_count WHERE counts.replyto IS NOT NULL")
	if err3 != nil {
		fmt.Println(err3)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err4 := db.Exec("UPDATE tweet t JOIN(SELECT retweetto, COUNT(*) AS retweet_count FROM tweet GROUP BY retweetto) AS counts ON t.id = counts.retweetto SET t.retweet = counts.retweet_count WHERE counts.retweetto IS NOT NULL")
	if err4 != nil {
		fmt.Println(err4)
		w.WriteHeader(http.StatusInternalServerError)
		return
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
}

func main() {
	// ② /userでリクエストされたらnameパラメーターと一致する名前を持つレコードをJSON形式で返す
	http.HandleFunc("/tweet", getTweet)
	http.HandleFunc("/getlike", getLike)
	http.HandleFunc("/like", toggleLike)
	http.HandleFunc("/follow", follow)
	http.HandleFunc("/followreq", followreq)
	http.HandleFunc("/gemini", askGemini)
	http.HandleFunc("/execute", executeOnGemini)
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
