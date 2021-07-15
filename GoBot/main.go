package main

import (
	"encoding/json"
	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/nu7hatch/gouuid"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var client redis.Client
var db *gorm.DB
var err error
//const token string = "1786001765:AAG40Y7K9MVfzDPriKRJHKetToDBx3T2V5s"
const token string = "1865007974:AAHuwFGJCb1AkVVlrxNVDFk_UZT-i4CvnKA"

const cloudinary_url string = "https://api.cloudinary.com/v1_1/demo/image/upload"
const cloudinary_API_KEY string = "896491813758597"
const cloudinary_API_SECRET string = "garf_hPU42GPUsyBAvaBQ9MjU2k"
const cloudinary_upload_preset string = "uxgityak"

type User struct {
	gorm.Model
	Tg_username string
	Full_name string
	Avatar string
	Lat float64
	Long float64
}

type respdecode struct {
	Secure_url string
}


func DBDefault()  {
	argstring := "host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable password=s6c89q4g"
	//argstring := "host=" + os.Getenv("ipv4addr") + " port=5432 user=postgres dbname=postgres sslmode=disable password=s6c89q4g"
	db, err = gorm.Open( "postgres", argstring)
	if err != nil {
		panic(err)
	}
}

func initCache() {
	// Initialize the redis connection to a redis instance running on your local machine
	client = *redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		//Addr: os.Getenv("ipv4addr") + ":6379",
		Password: "",
		DB: 0,
	})
}

func FindUser(usrname string) *User {
	DBDefault()
	defer db.Close()

	var usr User

	if err := db.Where("tg_username = ?", usrname).First(&usr).Error; err != nil {
		panic("what the hell")
	}
	return &usr

}

func UserExist(usrname string) bool {
	DBDefault()
	defer db.Close()

	var usr User
	if err := db.Where("tg_username = ?", usrname).First(&usr).Error; err != nil {
		return false
	}
	return true

}

func newUser(usr *User) {
	DBDefault()
	defer db.Close()

	if err := db.Create(usr).Error; err != nil {
		panic("error in newuser")
	}
}

func UpdateUser(usr *User) {
	DBDefault()
	defer db.Close()

	if err := db.Save(usr).Error; err != nil {
		panic("Error at UpdateUser")
	}
}

func setupuser(user *User) string{
	u, err := uuid.NewV4()
	if err != nil {
		println("uuid error", err)
	}
	json_data, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}
	err = client.Set(u.String(), json_data, time.Second*60).Err()
	if err != nil {
		panic(err)
	}
	err = client.Set(user.Tg_username, u.String(), time.Second*60).Err()
	if err != nil {
		panic(err)
	}
	return u.String()
}

func Message2User (updt *tgbotapi.Update, botptr *tgbotapi.BotAPI) *User {
	var LocalUser User;

	LocalUser.Full_name = updt.Message.From.FirstName + " " + updt.Message.From.LastName
	LocalUser.Tg_username = updt.Message.From.UserName
	res, err := botptr.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{UserID: updt.Message.From.ID})
	if err != nil{
		panic(err)
	}
	var photo string;
	if len(res.Photos) > 0 {
		tg_photo, err := botptr.GetFileDirectURL(res.Photos[0][0].FileID)
		if err != nil{
			panic(err)
		}
		resp, err := http.PostForm(cloudinary_url, url.Values{"file":{tg_photo},
			"api_key": {cloudinary_API_KEY},
			"upload_preset": {cloudinary_upload_preset},
		})
		if err != nil{
			panic(err)
		}
		newobj := respdecode{}
		json.NewDecoder(resp.Body).Decode(&newobj)
		photo = newobj.Secure_url

	} else {
		photo = "https://pmdoc.ru/wp-content/uploads/default-avatar-300x300.png"
	}
	LocalUser.Avatar = photo

	return &LocalUser
}

func main()  {
	initCache()
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	updateConfig := tgbotapi.NewUpdate(0)

	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		var user User;
		r, err := client.Exists(update.Message.From.UserName).Result()
		println("ressss", r)
		if err != nil {
			panic(err)
		}

		if r == 0 {
			if UserExist(update.Message.From.UserName) {
				println("Hello")
				user = *FindUser(update.Message.From.UserName)
				if strings.Contains(user.Avatar, "api.telegram.org") {
					freshuser := *Message2User(&update, bot)
					user.Avatar = freshuser.Avatar
					UpdateUser(&user)
				}
			} else {
				user = *Message2User(&update, bot)
				newUser(&user)
			}

		} else {
			prevtoken, err := client.Get(update.Message.From.UserName).Result()
			if err != nil {
				panic(err)
			}
			val, err := client.Get(prevtoken).Result()
			if err != nil {
				println("PUPIX", prevtoken)
				panic(err)
			}
			json.Unmarshal([]byte(val), &user);
			err = client.Del(prevtoken).Err()
			if err != nil {
				println("error deleting prevtoken", err)
			}
		}

		newtoken := setupuser(&user)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У тебя есть 1 минута, чтобы перейти по ссылке. http://167.99.234.173/view/?authtoken="+newtoken)
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}

		time.Sleep(5 * time.Second)

	}
}
