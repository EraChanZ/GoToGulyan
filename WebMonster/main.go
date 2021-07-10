package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/nu7hatch/gouuid"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

type User struct {
	gorm.Model
	Tg_username string
	Full_name string
	Avatar string
	Lat float64
	Long float64
}

var db *gorm.DB
var err error


type Page struct {
	Cur_user User
	Sec_code string
	Cookie_token string
	Authorized bool
	Registered bool
}


var client redis.Client

func LogIn(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		println("ParseForm err: %v", err)
		return
	}

	username := r.FormValue("tg_username")

	usrexist := UserExist(username);

	u, err := uuid.NewV4()
	if err != nil {
		println("uuid error", err)
		return
	}
	json_data, err := json.Marshal(Page{Cookie_token: u.String(), Sec_code: u.String()[:6], Cur_user: User{Tg_username: username}, Registered: usrexist})
	if err != nil {
		println("json error", err)
		return
	}

	err = client.Set(u.String(), json_data, time.Second*60*5).Err()
	if err != nil {
		println(err)
	}



	err = client.Set(username, json_data, time.Second*60*5).Err()
	if err != nil {
		println(err)
	}

	//_, err = cache.Do("SET", u.String(), "120", &(Page{Sec_code: u.String()[:6]}) )


	http.SetCookie(w, &http.Cookie{
		Name:    "Gotomeets_session_token",
		Value:   u.String(),
		Expires: time.Now().Add((60 * 60) * time.Second),
		Path: "/",
	})

	http.Redirect(w, r,"/view", http.StatusFound)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {

	t, _ := template.ParseFiles("view.html")

	c, err := r.Cookie("Gotomeets_session_token")
	if err != nil {
		println("maniga", c)
		if err == http.ErrNoCookie {
			p := &Page{}
			t.Execute(w, p)
			return
		}
	}

	val, err := client.Get(c.Value).Result()
	var curpage Page;
	if err != nil {
		println("redis recieve error", err)
	}
	json.Unmarshal([]byte(val), &curpage);
	println("bruh", curpage.Sec_code)

	if err != nil {
		// If there is an error fetching from cache, return an internal server error status
		log.Println("error retrieving token", err)
		return
	}

	t.Execute(w, curpage)
}


func setLocation(w http.ResponseWriter, r *http.Request){
	println("request to seloloc")
	if err := r.ParseForm(); err != nil {
		println("ParseForm err: %v", err)
		return
	}
	c, err := r.Cookie("Gotomeets_session_token")
	lat := r.FormValue("lat")
	long := r.FormValue("long")
	val, err := client.Get(c.Value).Result()
	if err != nil {
		println("redis recieve error", err)
		return
	}
	var curpage Page;
	json.Unmarshal([]byte(val), &curpage);

	curpage.Cur_user.Lat, err = strconv.ParseFloat(lat, 64)
	if err != nil {
		println("parse error")
		return
	}
	curpage.Cur_user.Long, err = strconv.ParseFloat(long, 64)

	if err != nil {
		println("parse error")
		return
	}

	UpdateUser(&curpage.Cur_user)

	json_data, err := json.Marshal(curpage)

	if err != nil {
		panic( err)
	}

	err = client.Set(curpage.Cookie_token, json_data, 0).Err()

	if err != nil {
		panic(err)
	}



}

func initCache() {
	// Initialize the redis connection to a redis instance running on your local machine
	client = *redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
		PoolSize: 100,
	})
}

func UserExist(usrname string) bool {
	db, err = gorm.Open( "postgres", "host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable password=s6c89q4g")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var usr User

	if err := db.Where("tg_username = ?", usrname).First(&usr).Error; err != nil {
		return false
	}
	return true

}

func UpdateUser(usr *User) {
	db, err = gorm.Open( "postgres", "host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable password=s6c89q4g")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Save(usr).Error; err != nil {
		panic("Error at UpdateUser")
	}

}

func initialMigration() {
	db, err = gorm.Open( "postgres", "host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable password=s6c89q4g")
	if err != nil {
		fmt.Println(err.Error())
		panic("failed to connect database")
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&User{})
}

func main() {

	initialMigration()
	initCache()
	http.HandleFunc("/login/", LogIn)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/setlocation/", setLocation)
	log.Fatal(http.ListenAndServe(":8080", nil))



}
