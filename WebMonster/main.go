package main

import (
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/nu7hatch/gouuid"
	"html/template"
	"log"
	"net/http"
	"os"
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

func (u User) AllOtherUsers() []User{
	DBDefault()
	defer db.Close()

	var users []User
	db.Find(&users)

	for indptr := 0;indptr < len(users);indptr++ {
		if users[indptr].Tg_username == u.Tg_username {
			users = append(users[:indptr], users[indptr+1:]...)
			break
		}
	}

	return users
}

var db *gorm.DB
var err error


type Page struct {
	Cur_user User
	Authorized bool
}


var client redis.Client

func LogIn(user *User, prevtoken string, w http.R

esponseWriter)  {

	u, err := uuid.NewV4()
	if err != nil {
		println("uuid error", err)
	}

	json_data, err := json.Marshal(*user)
	if err != nil {
		println("json error", err)
	}

	err = client.Set(u.String(), json_data, time.Second*60*60).Err()
	if err != nil {
		println("error setting cookie", err)
	}

	err = client.Del(prevtoken).Err()
	if err != nil {
		println("error deleting prevtoken", err)
	}

	err = client.Set(user.Tg_username, u.String(), time.Second*60*60).Err()
	if err != nil {
		println(err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "Gotomeets_session_token",
		Value:   u.String(),
		Expires: time.Now().Add((60 * 60) * time.Second),
		Path: "/",
	})


}

func viewHandler(w http.ResponseWriter, r *http.Request) {

	t, _ := template.ParseFiles("view.html")

	authtoken := r.URL.Query().Get("authtoken")
	if authtoken != "" {
		val, err := client.Get(authtoken).Result()
		if err == nil {
			var curuser User;
			json.Unmarshal([]byte(val), &curuser);
			LogIn(&curuser, authtoken, w)
			http.Redirect(w, r,"/", http.StatusFound)
			return
		}
	}

	c, err := r.Cookie("Gotomeets_session_token")
	if err != nil {
		p := &Page{}
		t.Execute(w, p)
		return
	}
	println("cookie", c.Value)
	val, err := client.Get(c.Value).Result()
	var curuser User;
	var curpage Page;
	if err != nil {
		println("yes redis recieve error")
	} else {
		err = json.Unmarshal([]byte(val), &curuser);
		if err == nil {
			curpage.Cur_user = curuser
			curpage.Authorized = true
		}
	}

	t.Execute(w, curpage)
}


func setLocation(w http.ResponseWriter, r *http.Request){
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
	var curuser User;
	json.Unmarshal([]byte(val), &curuser);
	println("request to seloloc was", curuser.Lat, curuser.Long)

	curuser.Lat, err = strconv.ParseFloat(lat, 64)
	if err != nil {
		println("parse error")
		return
	}
	curuser.Long, err = strconv.ParseFloat(long, 64)
	if err != nil {
		println("parse error")
		return
	}

	println("request to seloloc stal", curuser.Lat, curuser.Long)

	UpdateUser(&curuser)

	json_data, err := json.Marshal(curuser)


	if err != nil {
		panic( err)
	}
	err = client.Set(c.Value, json_data, time.Second * 60 * 60).Err()

	if err != nil {
		panic(err)
	}



}

func initCache() {
	// Initialize the redis connection to a redis instance running on your local machine
	client = *redis.NewClient(&redis.Options{
		//Addr: os.Getenv("ipv4addr") + ":6379",
		Addr: "127.0.0.1:6379",
		Password: "",
		DB: 0,
		PoolSize: 100,
	})
}

func DBDefault()  {
	//argstring := "host=" + os.Getenv("ipv4addr") + " port=5432 user=postgres dbname=postgres sslmode=disable password=s6c89q4g"
	argstring := "host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable password=" + os.Getenv("POSTGRES_PASSWORD")
	db, err = gorm.Open( "postgres", argstring)
	if err != nil {
		panic(err)
	}
}



func UpdateUser(usr *User) {
	DBDefault()
	defer db.Close()

	if err := db.Save(usr).Error; err != nil {
		panic("Error at UpdateUser")
	}

}

func initialMigration() {
	DBDefault()
	defer db.Close()
	// Migrate the schema
	db.AutoMigrate(&User{})
}

func main() {

	initialMigration()
	initCache()
	http.HandleFunc("/", viewHandler)
	http.HandleFunc("/setlocation/", setLocation)
	log.Fatal(http.ListenAndServe(":80", nil))

}
