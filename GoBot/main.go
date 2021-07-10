package main

import (
	"encoding/json"
	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var client redis.Client
var db *gorm.DB
var err error
const token string = "1786001765:AAG40Y7K9MVfzDPriKRJHKetToDBx3T2V5s"

type User struct {
	gorm.Model
	Tg_username string
	Full_name string
	Avatar string
	Lat float64
	Long float64
}

type Page struct {
	Cur_user User
	Sec_code string
	Cookie_token string
	Authorized bool
	Registered bool
}

func initCache() {
	// Initialize the redis connection to a redis instance running on your local machine
	client = *redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})
}

func FindUser(usrname string) *User {
	db, err = gorm.Open( "postgres", "host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable password=s6c89q4g")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var usr User

	if err := db.Where("tg_username = ?", usrname).First(&usr).Error; err != nil {
		panic("what the hell")
	}
	return &usr

}

func newUser(usr *User) {
	db, err = gorm.Open( "postgres", "host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable password=s6c89q4g")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Create(usr).Error; err != nil {
		panic("error in newuser")
	}


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
		r, err := client.Exists(update.Message.From.UserName).Result()
		if err != nil {
			panic(err)
		}
		if r == 1 {
			val, err := client.Get(update.Message.From.UserName).Result()
			if err != nil {
				panic(err)
			}

			var curpage Page;
			json.Unmarshal([]byte(val), &curpage);
			var msgtext string;

			if curpage.Sec_code == update.Message.Text {
				msgtext = "Код верный. Молодес"
				curpage.Authorized = true
				if curpage.Registered {
					println("I am at registered branch")
					reguser := *FindUser(curpage.Cur_user.Tg_username)
					println(reguser.Lat, reguser.Long)
					curpage.Cur_user = reguser
				} else {
					curpage.Cur_user.Full_name = update.Message.From.FirstName + " " + update.Message.From.LastName
					res, err := bot.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{UserID: update.Message.From.ID})
					if err != nil{
						panic(err)
					}
					photo, err := bot.GetFileDirectURL(res.Photos[0][0].FileID)
					curpage.Cur_user.Avatar = photo
					if err != nil{
						panic(err)
					}
					newUser(&curpage.Cur_user)
				}

				json_data, err := json.Marshal(curpage)
				if err != nil {
					panic( err)
				}
				err = client.Set(curpage.Cookie_token, json_data, 0).Err()
				if err != nil {
					panic(err)
				}
				err = client.Del(update.Message.From.UserName).Err()
				if err != nil {
					panic(err)
				}


			}else {
				msgtext = "Код НЕверный. НЕМолодес"
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgtext)
			if _, err := bot.Send(msg); err != nil {
				panic(err)
			}
		}
		/*
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}*/
	}
}
