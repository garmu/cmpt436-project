package webserver

import (
	"encoding/json"
	"github.com/gorilla/pat"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	configFilePath = ".config.json"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	//We shouldn't need a CheckOrigin function, because at this point the session is
	//already validated.
}

var router *pat.Router

func redirectHandler(conf *config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+conf.Website_url+conf.Https_portNum+r.RequestURI,
			http.StatusMovedPermanently)
	}
}

// handles websocket requests from the client.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	session := validateSessionAndLogInIfNecessary(w, r)
	if session == nil {
		return
	}

	log.Println("opening websocket")

	// may not need user info, but if we do, we can get it like this.
	// user, err := getUserFromSession(session)
	// if err != nil {
	// 	http.Error(w, "unable to retrieve user info",
	// 		http.StatusInternalServerError)
	// 	return
	// }
	// userName := user.Name

	ws, err := upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	serveWs(ws)
}

//To be used for any REST calls.
func writeJson(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Fatalln(err)
	}
}

//These will be marshaled directly from json
type config struct {
	Gplus          genericAuthConfig `json:"gplus"`
	Facebook       genericAuthConfig `json:"facebook"`
	Session_secret string            `json:"session_secret"`
	Website_url    string            `json:"website_url"`
	Https_portNum  string            `json:"https_portnum"`
	Http_portNum   string            `json:"http_portnum"`
}

func Serve() {
	configBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln("unable to read file ", configFilePath,
			":", err)
	}

	conf := &config{}
	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		log.Fatalln("unable to unmarshal config file:", err)
	}
	fm = NewFeedsManager()
	router = pat.New()
	router.Get("/ws", wsHandler)

	//This has to be the last thing called with the router, because it sets
	//the handler for the website root.
	initAuth(router, conf)
	http.Handle("/", router)
	//This static final can only be reached via explicit redirect: typing it into
	//the address bar just makes the router handle it.
	http.Handle("/app/", http.StripPrefix("/app/",
		http.FileServer(http.Dir("app/"))))

	go log.Fatal(http.ListenAndServeTLS(conf.Https_portNum,
		"cert.crt", "key.key", nil))
	log.Fatal(http.ListenAndServe(conf.Http_portNum,
		http.HandlerFunc(redirectHandler(conf))))
}
