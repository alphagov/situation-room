package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/Sirupsen/logrus"
)

var port = os.Getenv("PORT")

var calendarConfig = os.Getenv("MEETING_ROOM_CALENDARS")
var calendars map[string]string = make(map[string]string)

var client ApiClient
var googleApiKey = os.Getenv("MEETING_ROOM_API_KEY")
var googleClientId = os.Getenv("MEETING_ROOM_CLIENT_ID")

var authUsername = os.Getenv("MEETING_ROOM_AUTH_USER")
var authPassword = os.Getenv("MEETING_ROOM_AUTH_PASS")

var rooms map[string]Room = make(map[string]Room)

// This version can be set at build time via the -X flag.
var Version = "Not Set"

func main() {
	client = ApiClient{
		ClientId:   googleClientId,
		EncodedKey: googleApiKey,
	}
	calendars = parseCalendarConfig(calendarConfig)

	startTicker()

	logLevel := determineLogLevel()
	log.SetLevel(logLevel)

	log.Info("API is starting up on :" + port)
	log.Info("Use Ctrl+C to stop")

	http.HandleFunc("/rooms", roomsIndexHandler)
	http.HandleFunc("/rooms/", roomsShowHandler)
	http.HandleFunc("/_status", statusHandler)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func determineLogLevel() logrus.Level {
	var l logrus.Level
	l, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		l = logrus.ErrorLevel
	}

	return l
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	renderJSON(w, map[string]string{"version": Version})
}

func renderJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(v); err != nil {
		panic(err)
	}
}

func Authenticate(user, realm string) string {
	if user == authUsername {
		d := sha1.New()
		if _, err := d.Write([]byte(authPassword)); err != nil {
			panic(err)
		}
		e := base64.StdEncoding.EncodeToString(d.Sum(nil))

		return "{SHA}" + e
	}
	return ""
}

func roomsIndexHandler(w http.ResponseWriter, r *http.Request) {
	roomSet := RoomSet{
		Rooms:       rooms,
		TotalRooms:  len(calendars),
		RoomsLoaded: len(rooms),
	}
	apiResponse := RoomSetApiResponse{
		RoomSet: roomSet,
	}

	status := "ok"
	if !roomsLoaded() {
		status = "incomplete"
	}

	renderJSON(w, apiResponse.present(status))
}

func roomsShowHandler(w http.ResponseWriter, r *http.Request) {
	roomExp := regexp.MustCompile("^/rooms/([a-zA-Z0-9]+)$")
	dummyReq := http.Request{}

	var roomId string

	m := roomExp.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, &dummyReq)
		return
	} else {
		roomId = m[1]
	}

	room, ok := rooms[roomId]
	if !ok {
		http.NotFound(w, &dummyReq)
		return
	}

	apiResponse := RoomApiResponse{
		Room: room,
	}

	status := "ok"
	if !roomsLoaded() {
		status = "incomplete"
	}

	renderJSON(w, apiResponse.present(status))
}

func roomsLoaded() bool {
	if len(calendars) > len(rooms) {
		return false
	}
	return true
}

func loadEvents() {
	log.Debug("Loading events...")

	defer func() {
		if err := recover(); err != nil {
			log.Debug("loadEvents failed:", err)
		}
	}()

	client.Token = client.GetToken()

	for calendarName, calendarId := range calendars {
		go loadEventsForRoom(calendarName, calendarId)
	}
}

func loadEventsForRoom(calendarName string, calendarId string) {
	log.WithFields(log.Fields{
		"calendarName": calendarName,
	}).Debug("Start: Loading")
	startTime := time.Now()
	endTime := startTime.Truncate(24 * time.Hour).Add(24 * time.Hour)
	events, err := client.Api().Events.List(calendarId).
		TimeMin(startTime.Format(time.RFC3339)).
		TimeMax(endTime.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").Do()

	if err != nil {
		log.WithFields(log.Fields{
			"calendarName":  calendarName,
			logrus.ErrorKey: err,
		}).Error("Error loading")
	} else {
		rooms[calendarName] = CreateRoomFromEvents(calendarId, calendarName, events.Items)
		log.WithFields(log.Fields{
			"calendarName": calendarName,
			"eventCount":   len(rooms[calendarName].Events),
		}).Debug("Finish: loading")
	}
}

func parseCalendarConfig(config string) map[string]string {
	calendarMap := map[string]string{}
	lines := strings.Split(config, ";")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		name := parts[0]
		url := parts[1]

		calendarMap[name] = url
	}

	return calendarMap
}

func startTicker() {
	go loadEvents()
	ticker := time.NewTicker(60 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				loadEvents()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
