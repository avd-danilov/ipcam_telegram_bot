package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// var NewDirs chan os.DirEntry
var StoreDir string
var TgApiToken string
var TgApiChatId int64
var FfmpegVideoArgs []string
var FfmpegScrShotArgs []string
var StoreBuf string
var TgBot *tgbotapi.BotAPI

type OsHandler struct {
	Caption string
	fPath   string
}

func (s *OsHandler) ReceiveVideo() error {
	timeVideo := time.Now().Format("02-01-2006--15-04-05")
	s.Caption = fmt.Sprintf("video%s.mp4", timeVideo)
	s.fPath = fmt.Sprintf("%svideo%s.mp4", StoreBuf, timeVideo)
	out := exec.Command("ffmpeg")
	out.Args = append(out.Args, FfmpegVideoArgs...)
	out.Args = append(out.Args, s.fPath)
	//out.Stdout = os.Stdout
	//out.Stdin = os.Stdin
	//out.Stderr = os.Stderr
	err := out.Run()
	return err
}
func TgSendText(text string) {
	if len(text) == 0 {
		return
	}
	msg := tgbotapi.NewMessage(TgApiChatId, text)
	_, err := TgBot.Send(msg)
	if err != nil {
		log.Println("Send text to telegram error: ", err)
	}
}
func TgSendVideo(s OsHandler) {
	msgVideo := tgbotapi.NewVideo(TgApiChatId, tgbotapi.FilePath(s.fPath))
	msgVideo.Caption = s.Caption
	_, err := TgBot.Send(msgVideo)
	if err != nil {
		log.Println("Send video to telegram error: ", err)
	}

}

func TgSendPhoto(s OsHandler) {
	msg := tgbotapi.NewPhoto(TgApiChatId, tgbotapi.FilePath(s.fPath))
	msg.Caption = s.Caption
	_, err := TgBot.Send(msg)
	if err != nil {
		log.Println("Send photo to telegram error: ", err)
	}
}
func (s *OsHandler) Delete() error {
	out := exec.Command("rm", "-r", s.fPath)
	err := out.Run()
	return err
}

func (s *OsHandler) Rename(newName string) error {
	indxFile := strings.LastIndex(s.fPath, "/")
	newfPath := fmt.Sprint(s.fPath[:indxFile], newName)
	out := exec.Command("mv", s.fPath, newfPath)
	err := out.Run()
	return err
}

func (s *OsHandler) ReceivePhoto() error {
	timePhoto := time.Now().Format("02-01-2006--15-04-05")
	s.fPath = fmt.Sprintf("%simg%s.jpeg", StoreBuf, timePhoto)
	out := exec.Command("ffmpeg")
	out.Args = append(out.Args, FfmpegScrShotArgs...)
	out.Args = append(out.Args, s.fPath) //добавим адрес и имя сохраняемого файла
	//out.Stdout = os.Stdout
	//out.Stdin = os.Stdin
	//out.Stderr = os.Stderr
	err := out.Run()
	return err
}

//func (s *OsHandler)  ReceiveLogs(t time.Time) error {
//
//	return nil
//}

func readStore() {
	//tLastScan := time.Date(1970, 01, 01, 0, 0, 0, 0, time.Local) //время создания последнего файла за сканирование
	//var tNewScan time.Time                                       //время самой новой папки в текущем сканировании

	var s OsHandler
	for { // Сканируем папку хранилища
		allDirs, err := os.ReadDir(StoreDir)
		if err != nil {
			log.Printf("Error read dir, %s", err.Error())
			return
		}
		currDate := time.Now().Format("20060102")
		for _, currDir := range allDirs {
			if currDir.IsDir() { //Это папка?
				currInfoDir, err := currDir.Info() //Получим информацию о папке
				if err != nil {
					log.Printf("Read file info error, %s", err.Error())
					return
				}
				if time.Since(currInfoDir.ModTime()).Hours() > 720 { //удалим папку если старая
					s.fPath = fmt.Sprintf("%s%s", StoreDir, currDir.Name())
					err := s.Delete()
					if err != nil {
						log.Printf("Err delete dir: %s, %s", s.fPath, err.Error())
						//return
					}
					continue
				}

				if currInfoDir.Name() == currDate { //search today dir
					pFiles := fmt.Sprintf("%s%s/picture", StoreDir, currDate)
					allFiles, err := os.ReadDir(pFiles) //прочитаем файлы из этой папки
					if err != nil {                     //if no file`s in directory
						if !strings.Contains(err.Error(), "no such file or directory") {
							continue
						}
						log.Printf("Err read dir: %s, %s", pFiles, err.Error())
						return
					}
					for _, currFile := range allFiles {

						if !currFile.IsDir() && strings.Contains(currFile.Name(), ".jpeg") &&
							!strings.Contains(currFile.Name(), "[POSTED].jpeg") {
							s.fPath = fmt.Sprintf("%s%s/picture/%s", StoreDir, currDate, currFile.Name())
							s.Caption = currFile.Name()
							TgSendPhoto(s)
							//newName := currFile.Name()
							newName := strings.TrimRight(currFile.Name(), ".jpeg")
							newName = fmt.Sprintf("%s[POSTED].jpeg", newName)
							err := os.Rename(s.fPath, fmt.Sprintf("%s%s/picture/%s", StoreDir, currDate, newName))
							if err != nil {
								log.Printf("Error rename file %s:  %s", newName, err.Error())
								return
							}

						}
					}

				}

				//if currInfoDir.ModTime().After(DirModTime) { //Папка создана после последнего сканирования?
				//	i <- currDir
				//	if currInfoDir.ModTime().After(newDirModTime) { //Поиск самой новой папки
				//		newDirModTime = currInfoDir.ModTime()
				//	}
				//
				//}
			}
		}
		time.Sleep(1000 * time.Millisecond)
	}

}

func tgCommandHandler() error {
	var tgUpdates tgbotapi.UpdatesChannel
	var err error
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	tgUpdates = TgBot.GetUpdatesChan(u)

	for update := range tgUpdates {
		var osHand OsHandler
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			switch update.Message.Text {
			case "video", "Video", "Видео", "видео":
				TgSendText("Начало записи... 30 сек")
				err = osHand.ReceiveVideo()
				if err != nil {
					TgSendText(err.Error())
				}

				TgSendVideo(osHand)

				err = osHand.Delete()
				if err != nil {
					TgSendText(err.Error())
				}

			case "photo", "Photo", "Фото", "фото":
				TgSendText("Начало сохранения... 1 кадр")
				err = osHand.ReceivePhoto()
				if err != nil {
					TgSendText(err.Error())
				}
				TgSendPhoto(osHand)
				err = osHand.Delete()
				if err != nil {
					TgSendText(err.Error())
				}
			default:
				TgSendText("Not recognized command")
			}
			//msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			//msg.ReplyToMessageID = update.Message.MessageID
			//
			//TgBot.Send(msg)
			//
			//msgphoto := tgbotapi.NewVideo(update.Message.Chat.ID, tgbotapi.FilePath("./video.mp4"))
			//TgBot.Send(msgphoto)
		}
	}
	return err
}
func main() {
	//Initialization
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	log.SetOutput(file)

	//NewDirs = make(chan os.DirEntry, 30) //канал для передачи имени папки

	err = godotenv.Load("conf.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	TgApiToken = os.Getenv("API_TG_BOT")
	StoreDir = os.Getenv("STORAGE_DIR")
	FfmpegVideoArgs = strings.Split(os.Getenv("UTILS_FFMPEG_VIDEO_ARGS"), " ")
	FfmpegScrShotArgs = strings.Split(os.Getenv("UTILS_FFMPEG_SSHOT_ARGS"), " ")
	StoreBuf = os.Getenv("STORAGE_BUF")
	val, err := strconv.Atoi(os.Getenv("API_TG_BOT_CHAT_ID"))
	if err != nil {
		return
	}
	TgApiChatId = int64(val)

	//Start telegram bot
	TgBot, err = tgbotapi.NewBotAPI(TgApiToken)
	if err != nil {
		log.Panic(err)
	}
	TgBot.Debug = true
	log.Printf("Authorized on account %s", TgBot.Self.UserName)
	TgSendText("Bot started")

	//Start commands handler
	go func() {
		err := tgCommandHandler()
		if err != nil {
			log.Println(err)
		}
	}()
	//Start
	readStore()

}
