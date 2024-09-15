package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var NewDirs chan os.DirEntry = make(chan os.DirEntry, 30) //канал для передачи имени папки
// var TgbotUpdate tgbotapi.UpdatesChannel
var StoreDir string
var TgApiToken string
var FfmpegVideoArgs []string
var FfmpegSshotArgs []string

func readStore() {

	shed := func(i chan os.DirEntry) {
		var DirModTime time.Time    //время создания последней папки за все сканирование
		var newDirModTime time.Time //время самой новой папки в текущем сканировании
		DirModTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.Local)
		for { // Сканируем папки директории
			allDirs, err := os.ReadDir(StoreDir)
			if err != nil {
				return
			}

			for _, currDir := range allDirs {
				if currDir.IsDir() { //Это папка?
					currDateDir, err := currDir.Info() //Получим информацию
					if err != nil {
						return
					}
					if currDateDir.ModTime().After(DirModTime) { //Папка создана после последнего сканирования?
						i <- currDir
						if currDateDir.ModTime().After(newDirModTime) { //Поиск самой новой папки
							newDirModTime = currDateDir.ModTime()
						}

					}
				}
			}
			DirModTime = newDirModTime // Запишем самое последнее изменение папок после последнего сканирования

			time.Sleep(100 * time.Millisecond)
		}

	}

	go shed(NewDirs)
}

func main() {
	err := godotenv.Load("conf.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	TgApiToken = os.Getenv("API_TG_BOT")
	StoreDir = os.Getenv("STORAGE_DIR")
	FfmpegVideoArgs = strings.Split(os.Getenv("UTILS_FFMPEG_VIDEO_ARGS"), " ")
	FfmpegSshotArgs = strings.Split(os.Getenv("UTILS_FFMPEG_SSHOT_ARGS"), " ")
	storeBuf := os.Getenv("STORAGE_BUF")
	fmt.Println(FfmpegVideoArgs)
	go readStore()
	var tgUpdates tgbotapi.UpdatesChannel
	bot, err := tgbotapi.NewBotAPI(TgApiToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	tgUpdates = bot.GetUpdatesChan(u)
	for update := range tgUpdates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			switch update.Message.Text {
			case "video", "Video", "Видео", "видео":
				msgPre := tgbotapi.NewMessage(update.Message.Chat.ID, "Начало записи... 30 сек")
				_, err = bot.Send(msgPre)
				if err != nil {
					return
				}

				timeVideo := time.Now().Format("02-01-2006--15-04-05")
				pathVideo := fmt.Sprintf("%svideo%s.mp4", storeBuf, timeVideo)
				out := exec.Command("ffmpeg")
				out.Args = append(out.Args, FfmpegVideoArgs...)
				out.Args = append(out.Args, pathVideo)
				//out.Stdout = os.Stdout
				//out.Stdin = os.Stdin
				//out.Stderr = os.Stderr
				err = out.Run()
				if err != nil {
					msgErr := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка ffmpeg: %s", err))
					_, err = bot.Send(msgErr)
					if err != nil {
						return
					}
				}
				msgVideo := tgbotapi.NewVideo(update.Message.Chat.ID, tgbotapi.FilePath(pathVideo))
				msgVideo.Caption = fmt.Sprintf("video%s.mp4", timeVideo)
				_, err = bot.Send(msgVideo)
				if err != nil {
					return
				}
				out = exec.Command("rm", pathVideo)
				err = out.Run()
				if err != nil {
					msgErr := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err))
					_, err = bot.Send(msgErr)
					if err != nil {
						return
					}
				}
			case "photo", "Photo", "Фото", "фото":
				msgPre := tgbotapi.NewMessage(update.Message.Chat.ID, "Начало сохранения... 1 кадр")
				_, err = bot.Send(msgPre)
				if err != nil {
					return
				}
				timePhoto := time.Now().Format("02-01-2006--15-04-05")
				pathPhoto := fmt.Sprintf("%simg%s.jpeg", storeBuf, timePhoto)
				out := exec.Command("ffmpeg")
				out.Args = append(out.Args, FfmpegSshotArgs...)
				out.Args = append(out.Args, pathPhoto) //добавим адрес и имя сохраняемого файла
				//out.Stdout = os.Stdout
				//out.Stdin = os.Stdin
				//out.Stderr = os.Stderr
				err = out.Run()
				if err != nil {
					msgErr := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка ffmpeg: %s", err))
					_, err = bot.Send(msgErr)
					if err != nil {
						return
					}
				}
				msgPhoto := tgbotapi.NewPhoto(update.Message.Chat.ID, tgbotapi.FilePath(pathPhoto))
				msgPhoto.Caption = fmt.Sprintf("img%s.jpeg", timePhoto)
				_, err = bot.Send(msgPhoto)
				if err != nil {
					return
				}
				out = exec.Command("rm", pathPhoto)
				err = out.Run()
				if err != nil {
					msgErr := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err))
					_, err = bot.Send(msgErr)
					if err != nil {
						return
					}
				}
			}
			//msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			//msg.ReplyToMessageID = update.Message.MessageID
			//
			//bot.Send(msg)
			//
			//msgphoto := tgbotapi.NewVideo(update.Message.Chat.ID, tgbotapi.FilePath("./video.mp4"))
			//bot.Send(msgphoto)
		}
	}

	//fmt.Println(dir)
	for cdir := range NewDirs {
		if cdir.IsDir() {
			inf, _ := cdir.Info()
			fmt.Println(cdir.Name(), "Info:", inf.ModTime())
		}
	}

}
