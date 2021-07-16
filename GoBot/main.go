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

var Markups = map[string]tgbotapi.ReplyKeyboardMarkup{
	"/" : tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Клик"),
			tgbotapi.NewKeyboardButton("Обновить данные"),
		),
	),
	"/pd/" : tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Фотку"),
			tgbotapi.NewKeyboardButton("Полное имя"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	),
	"/pd/photo/" : tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	),
	"/pd/fullname/": tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	),
}


var pd_photo = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Назад"),
	),
)

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

var BotSessions = make(map[int]string)


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

func reducepath(s string) string{
	arr := strings.Split(s, "/")
	return strings.Join(arr[:len(arr)-2], "/") + "/"
}

func FullUpdateUser(usr *User)  {
	curtoken, err := client.Get(usr.Tg_username).Result()
	if err != nil {
		panic(err)
	}
	json_data, err := json.Marshal(usr)
	if err != nil {
		panic(err)
	}
	expires_in, err := client.TTL(curtoken).Result()
	if err != nil {
		panic(err)
	}
	client.Set(curtoken, json_data, expires_in)
	UpdateUser(usr)
}

func setupuser(user *User, prevtoken string) string{
	if prevtoken != "" {
		err = client.Del(prevtoken).Err()
		if err != nil {
			println("error deleting prevtoken", err)
		}
	}
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
		var prevtoken string;

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
			prevtoken, err = client.Get(update.Message.From.UserName).Result()
			if err != nil {
				panic(err)
			}
			val, err := client.Get(prevtoken).Result()
			if err != nil {
				panic(err)
			}
			json.Unmarshal([]byte(val), &user)

		}

		if _, ok := BotSessions[update.Message.From.ID]; !ok {
			BotSessions[update.Message.From.ID] = "/"
		}

		var response tgbotapi.MessageConfig;

		switch BotSessions[update.Message.From.ID] {
		case "/":
			switch update.Message.Text {
			case "Обновить данные":
				BotSessions[update.Message.From.ID] += "pd/"
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "Что хочешь обновить?")
			default:
				newtoken := setupuser(&user, prevtoken)
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "У тебя есть 1 минута, чтобы перейти по ссылке. http://167.99.234.173/view/?authtoken="+newtoken)
			}
		case "/pd/":
			switch update.Message.Text {
			case "Назад":
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "Вернулся")
				BotSessions[update.Message.From.ID] = reducepath(BotSessions[update.Message.From.ID])
			case "Фотку":
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "Присылай фотку")
				BotSessions[update.Message.From.ID] += "photo/"
			case "Полное имя":
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "Пиши полное имя")
				BotSessions[update.Message.From.ID] += "fullname/"
			default:
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "Че ты не то сказал")
			}
		case "/pd/photo/":
			switch update.Message.Text {
			case "Назад":
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "Вернулся")
				BotSessions[update.Message.From.ID] = reducepath(BotSessions[update.Message.From.ID])
			default:
				if len(update.Message.Photo) > 0 {
					tg_photo, err := bot.GetFileDirectURL(update.Message.Photo[0].FileID)
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
					err = json.NewDecoder(resp.Body).Decode(&newobj)
					if err != nil{
						panic(err)
					}
					user.Avatar = newobj.Secure_url
					FullUpdateUser(&user)
					response = tgbotapi.NewMessage(update.Message.Chat.ID, "Загрузил фотку, возвращаю назад.")
					BotSessions[update.Message.From.ID] = reducepath(BotSessions[update.Message.From.ID])
				} else {
					response = tgbotapi.NewMessage(update.Message.Chat.ID, "Ты ниче не прислал")
				}
			}
		case "/pd/fullname/":
			switch update.Message.Text {
			case "Назад":
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "Вернулся")
				BotSessions[update.Message.From.ID] = reducepath(BotSessions[update.Message.From.ID])
			case "":
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "В твоём сообщении нет текста")
			default:
				user.Full_name = update.Message.Text
				FullUpdateUser(&user)
				response = tgbotapi.NewMessage(update.Message.Chat.ID, "Сохранил, возвращаю назад.")
				BotSessions[update.Message.From.ID] = reducepath(BotSessions[update.Message.From.ID])
			}
			}

		println(BotSessions[update.Message.From.ID])
		response.ReplyMarkup = Markups[BotSessions[update.Message.From.ID]]
		if _, err := bot.Send(response); err != nil {
			panic(err)
		}

		time.Sleep(5 * time.Second)

	}
}
