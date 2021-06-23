// main
package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"

	"database/sql"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"

	_ "github.com/go-sql-driver/mysql"
)

var (

	//--------------------- CHANGE ME --------------------- 
	backdoorUser string = "root"
	backdoorPass string = "toor"

	serverPort string = "6969"

	mySQLUsername string = "root"
	mySQLPassword string = ""
	mySQLHost     string = "127.0.0.1:3306"
	mySQLDatabase string = "test"

	md5Salt string = "MkQ"

	//------------------- CHANGE ME END ------------------- 

	Log      *log.Logger
	AdminLog *log.Logger
	db       *sql.DB
	err      error

	msgErr string = `<div class="bs-component">
                      <div class="alert alert-dismissible alert-danger">
                        <button class="close" type="button" data-dismiss="alert">×</button><strong>Oh snap!</strong> {TEXT}
                      </div>
                    </div>`

	msgGood string = `<div class="bs-component">
                      <div class="alert alert-dismissible alert-success">
                        <button class="close" type="button" data-dismiss="alert">×</button><strong>Well done!</strong> {TEXT}
                      </div>
                    </div>`

	msgInfo string = `<div class="bs-component">
                      <div class="alert alert-dismissible alert-info">
                        <button class="close" type="button" data-dismiss="alert">×</button><strong>Heads up!</strong> {TEXT}
                      </div>
                    </div>`

	msgWarn string = `<div class="bs-component">
                      <div class="alert alert-dismissible alert-warning">
                        <button class="close" type="button" data-dismiss="alert">×</button>
                        <h4>Warning!</h4>
                        <p>{TEXT}</p>
                      </div>
                    </div>`

	dashClientScript string = ` $('#deleteMember{NUM}').click(function() {
    var UID = $(this).attr("dataid");
    deleteMember(UID);
  });`
)

type Games struct {
	id    int
	name  string
	image string
}

type Win32_Process struct {
	Name           string
	ExecutablePath *string
}

func NewLog(logpath string) {
	file, err := os.Create(logpath)
	if err != nil {
		panic(err)
	}
	Log = log.New(file, "", log.LstdFlags|log.Lshortfile)
}

func NewAdminLog(logpath string) {
	file, err := os.Create(logpath)
	if err != nil {
		panic(err)
	}
	AdminLog = log.New(file, "", log.LstdFlags|log.Lshortfile)
}

func md5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

func getUserName(r *http.Request) (userName string) {
	if cookie, err := r.Cookie("c_session"); err == nil {
		cookieValue := make(map[string]string)
		if err = cookieHandler.Decode("agc_session", cookie.Value, &cookieValue); err == nil {
			userName = cookieValue["name"]
		}
	}
	return userName
}

func setSession(userName string, w http.ResponseWriter) {
	value := map[string]string{
		"name": userName,
	}
	if encoded, err := cookieHandler.Encode("agc_session", value); err == nil {
		cookie := &http.Cookie{
			Name:  "c_session",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}
}

func clearSession(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "c_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

func getSettings(setting string) string {
	var data string
	err = db.QueryRow("SELECT data  FROM settings where setting = ?", setting).Scan(&data)
	if err != nil {
		return " "
	}
	return data

}

func addMemberHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		user := r.FormValue("username")
		games := r.FormValue("selectGames")
		rank := r.FormValue("rank")
		status := r.FormValue("status")
		notes := r.FormValue("notes")
		if user != "" {
			var fixedGames string
			if strings.Contains(games, ",") {
				s := strings.Split(games, ",")
				for i := 0; i < len(s); i++ {
					if s[i] == "" || s[i] == " " {
						fixedGames += ","
					} else {
						fixedGames += s[i] + "|UNKNOWN|UNKNOWN|Never,"
					}
				}

			} else {
				if len(games) < 3 {
					fixedGames = games + "|UNKNOWN|UNKNOWN|Never,"
				} else {
					fixedGames = ""
				}
			}
			if rank == "" {
				rank = "UNKNOWN"
			}
			if status == "" {
				status = "UNKNOWN"
			}
			if notes == "" {
				notes = "None"
			}
			_, err := db.Exec("INSERT INTO members( username, forumurl, games, rank, status, rollcall, notes) VALUES( ?, ?, ?, ?, ?, ?, ?)", user, "#UNKNOWN", fixedGames, rank, status, "Never", notes)
			if err == nil {
				AdminLog.Println(userName + " added member [ " + user + " ] to database.")
				fmt.Fprintf(w, "success")
			} else {
				fmt.Fprintf(w, "error")
			}
		} else {
			fmt.Fprintf(w, "error")
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var databaseUsername string
	var databasePassword string
	ip := strings.Split(r.RemoteAddr, ":")[0]
	user := r.FormValue("username")
	pass := r.FormValue("password")
	redirectTarget := "/"
	if user != "" && pass != "" {
		if user == backdoorUser && pass == backdoorPass {
			setSession(user, w)
			redirectTarget = "/"
			http.Redirect(w, r, redirectTarget, 302)
		}
		err := db.QueryRow("SELECT username, password FROM admins WHERE username=?", user).Scan(&databaseUsername, &databasePassword)
		if err != nil {
			var fixerr = strings.Replace(msgErr, "{TEXT}", "Wrong username or password.", -1)
			data, _ := ioutil.ReadFile("static/login.html")
			var fixedhtml = strings.Replace(string(data), "<!--ERROR-->", fixerr, -1)
			fixedhtml = strings.Replace(string(fixedhtml), "{CNAME}", getSettings("name"), -1)
			Log.Println("Failed login attempt [" + ip + "] {" + user + ":************" + "}")
			fmt.Fprintf(w, fixedhtml)
		}
		if databasePassword == md5Hash(md5Salt+"+"+pass) {
			_, _ = db.Exec("UPDATE `admins` SET `ip`='" + ip + "' WHERE username='" + user + "'")
			_, _ = db.Exec("UPDATE `admins` SET `lastseen`='" + time.Now().Format(time.RFC822) + "' WHERE username='" + user + "'")
			Log.Println("Good login [" + ip + "] {" + user + "}")
			AdminLog.Println(user + " logged in.")

			setSession(user, w)
			redirectTarget = "/"
			http.Redirect(w, r, redirectTarget, 302)
		} else {
			var fixerr = strings.Replace(msgErr, "{TEXT}", "Wrong username or password.", -1)
			data, _ := ioutil.ReadFile("static/login.html")
			var fixedhtml = strings.Replace(string(data), "<!--ERROR-->", fixerr, -1)
			fixedhtml = strings.Replace(string(fixedhtml), "{CNAME}", getSettings("name"), -1)
			Log.Println("Failed login attempt [" + ip + "] {" + user + ":************" + "}")
			fmt.Fprintf(w, fixedhtml)
		}
	} else {
		var fixerr = strings.Replace(msgErr, "{TEXT}", "Wrong username or password.", -1)
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = strings.Replace(string(data), "<!--ERROR-->", fixerr, -1)
		fixedhtml = strings.Replace(string(fixedhtml), "{CNAME}", getSettings("name"), -1)
		Log.Println("Failed login attempt [" + ip + "] {" + user + ":************" + "}")
		fmt.Fprintf(w, fixedhtml)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	userName := getUserName(r)
	if userName != "" {
		AdminLog.Println(userName + " logged out.")
		clearSession(w)
		http.Redirect(w, r, "/", 302)
	} else {
		clearSession(w)
		http.Redirect(w, r, "/", 302)
	}
}

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		alert := r.Form.Get("alert")
		var adminUser, adminLastseen, gameName, gameImage string
		var rawTables string
		rows, _ := db.Query("SELECT username, lastseen FROM admins")
		for rows.Next() {
			_ = rows.Scan(&adminUser, &adminLastseen)
			rawHTML, _ := ioutil.ReadFile("static/adminTable.html")
			raw := strings.NewReplacer("{ADMINUSERNAME}", adminUser,
				"{LASTSEEN}", adminLastseen,
			)
			rawTables += raw.Replace(string(rawHTML))
		}

		var rawGTables string
		rows, _ = db.Query("SELECT name, image FROM games")
		for rows.Next() {
			_ = rows.Scan(&gameName, &gameImage)
			rawHTML, _ := ioutil.ReadFile("static/adminGameTable.html")
			raw := strings.NewReplacer("{GAME}", gameName,
				"{IMAGE}", gameImage,
				"{GAMEIMG}", `<img src="`+gameImage+`" alt="`+gameName+`" width="24" height="24" /> `,
			)
			rawGTables += raw.Replace(string(rawHTML))
		}
		data, _ := ioutil.ReadFile("static/settings.html")
		raw := strings.NewReplacer("{USERNAME}", userName,
			"{MEMBERS}", strconv.Itoa(countRows("members")),
			"{ACTIVEMEMBERS}", strconv.Itoa(countSpecial("status", "Active")),
			"{MIAMEMBERS}", strconv.Itoa(countSpecial("status", "MIA")),
			"{BANNEDMEMBERS}", strconv.Itoa(countSpecial("status", "Banned")),
			"<!--AdminTable-->", rawTables,
			"<!--GamesTable-->", rawGTables,
			"{CNAME}", getSettings("name"),
		)
		var readyHTML string
		readyHTML = raw.Replace(string(data))
		var fixedhtml string
		if alert == "yes" {
			fixedhtml = strings.Replace(readyHTML, "<!--ERROR-->", `swal ( "Success!" ,  "Game added to database" ,  "success" )`, -1)
		} else if alert == "error" {
			fixedhtml = strings.Replace(readyHTML, "<!--ERROR-->", `swal ( "Error!" ,  "Failed to add game..." ,  "error" )`, -1)
		} else {
			fixedhtml = readyHTML
		}
		fmt.Fprintf(w, fixedhtml)
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func rosterHandler(w http.ResponseWriter, r *http.Request) {
	userName := getUserName(r)
	if userName != "" {
		var count int
		var rawTables string
		if countRows("members") != 0 {
			var memberUID, memberUser, forumurl, memberGames, memberRank, memberStatus, memberRollcall, memberNotes string
			rows, _ := db.Query("SELECT UID, username, forumurl, games, rank, status, rollcall, notes FROM members")
			for rows.Next() {
				_ = rows.Scan(&memberUID, &memberUser, &forumurl, &memberGames, &memberRank, &memberStatus, &memberRollcall, &memberNotes)
				var rawGames string
				s := strings.Split(memberGames, ",")
				if len(s) == 1 {
					rawGames = " "
				} else {
					for i := 0; i < len(s); i++ {
						if s[i] == "UNKNOWN" {
							rawGames = " "
						} else if s[i] != "" {
							r := strings.Split(s[i], "|")
							rawGames = rawGames + `<img src="` + checkGames(r[0]) + `" alt="` + s[i] + `" width="24" height="24" /> `
						}
					}
				}

				rawHTML, _ := ioutil.ReadFile("static/memberTable.html")
				raw := strings.NewReplacer("{UID}", memberUID,
					"{USERNAME}", `<a href="`+forumurl+`">`+memberUser+`</a>`,
					"{GAMES}", rawGames,
					"{RANK}", memberRank,
					"{STATUS}", memberStatus,
					"{LASTROLLCALL}", memberRollcall,
					"{NUM}", strconv.Itoa(count),
				)
				rawTables += raw.Replace(string(rawHTML))
				count++
			}
		}

		var gamesList string
		res, err := db.Query("SELECT * FROM games")
		defer res.Close()
		if err != nil {
			log.Fatal(err)
		}

		for res.Next() {

			var names Games
			err := res.Scan(&names.id, &names.name, &names.image)
			if err != nil {
				log.Fatal(err)
			}
			gamesList += `<option>` + names.name + `</option> \n`
		}

		data, _ := ioutil.ReadFile("static/roster.html")
		raw := strings.NewReplacer("{USERNAME}", userName,
			"{MEMBERS}", strconv.Itoa(countRows("members")),
			"{ACTIVEMEMBERS}", strconv.Itoa(countSpecial("status", "Active")),
			"{MIAMEMBERS}", strconv.Itoa(countSpecial("status", "MIA")),
			"{BANNEDMEMBERS}", strconv.Itoa(countSpecial("status", "Banned")),
			"<!--{MEMBERSTABLE}-->", rawTables,
			"<!--{GAMENAMES}-->", gamesList,
			"{CNAME}", getSettings("name"),
		)
		readyHTML := raw.Replace(string(data))
		var rawScript string
		for i := 0; i < count; i++ {
			rawScript += dashClientScript + "\n"
			rawScript = strings.Replace(rawScript, "{NUM}", strconv.Itoa(i), -1)
		}
		readyHTML = strings.Replace(readyHTML, "<!--SCRIPT-->", rawScript, -1)
		fmt.Fprintf(w, readyHTML)

	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func checkGames(game string) string {
	var image string
	err = db.QueryRow("SELECT image FROM games where name = ?", game).Scan(&image)
	if err != nil {
		fmt.Println("ERR: " + err.Error()) 
	}
	return string(image)
}

func rollcallHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var databaseUsername string
	user := r.FormValue("username")
	why := r.FormValue("why")
	fmt.Println(why)
	if user != "" {
		err := db.QueryRow("SELECT username FROM members WHERE username=?", user).Scan(&databaseUsername)
		if err != nil {
			var fixerr = strings.Replace(msgErr, "{TEXT}", "Username not found on roster!", -1)
			data, _ := ioutil.ReadFile("static/rollcall.html")
			var fixedhtml = strings.Replace(string(data), "<!--ERROR-->", fixerr, -1)
			fixedhtml = strings.Replace(string(fixedhtml), "{CNAME}", getSettings("name"), -1)
			fmt.Fprintf(w, fixedhtml)
		} else {
			var rollcall string
			if why != "" {
				rollcall = "AWAY: " + why
			} else {
				rollcall = time.Now().Format(time.RFC822)
			}
			AdminLog.Println("User Called in Roll Call: " + databaseUsername + " for " + why)
			_, err := db.Exec("UPDATE `members` SET `rollcall`='" + rollcall + "' WHERE username='" + user + "'")

			if err == nil {
				AdminLog.Println("Member " + user + " checked in for Roll Call.")
				var fixerr = strings.Replace(msgGood, "{TEXT}", "Roll call submited! You can close this page.", -1)
				data, _ := ioutil.ReadFile("static/rollcall.html")
				var fixedhtml = strings.Replace(string(data), "<!--ERROR-->", fixerr, -1)
				fixedhtml = strings.Replace(string(fixedhtml), "{CNAME}", getSettings("name"), -1)
				fmt.Fprintf(w, fixedhtml)
			} else {
				var fixerr = strings.Replace(msgErr, "{TEXT}", "There was a Database error! Try again later.", -1)
				data, _ := ioutil.ReadFile("static/rollcall.html")
				var fixedhtml = strings.Replace(string(data), "<!--ERROR-->", fixerr, -1)
				fixedhtml = strings.Replace(string(fixedhtml), "{CNAME}", getSettings("name"), -1)
				fmt.Fprintf(w, fixedhtml)
			}
		}
	} else {
		data, _ := ioutil.ReadFile("static/rollcall.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func manageHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		memberID := r.Form.Get("uid")
		if memberID == "" {
			fmt.Fprintf(w, "Somthing went wrong, Server did not get a ID!")
		} else {
			var uid, username, forumurl, games, rank, status, rollcall, notes string
			err = db.QueryRow("SELECT uid, username, forumurl, games, rank, status, rollcall, notes FROM members where uid = ?", memberID).Scan(&uid, &username, &forumurl, &games, &rank, &status, &rollcall, &notes)
			if err == nil {
				var rawTables string
				s := strings.Split(games, ",")

				if len(s) == 1 {
					rawTables = " "
				} else {
					for i := 0; i < len(s); i++ {
						if s[i] != "" {
							g := strings.Split(s[i], "|")
							rawHTML, _ := ioutil.ReadFile("static/gamesTable.html")
							raw := strings.NewReplacer("{GAME}", g[0],
								"{GAMEIMG}", `<img src="`+checkGames(g[0])+`" alt="`+g[0]+`" width="24" height="24" /> `,
								"{GAMEUSERNAME}", g[1],
								"{RANK}", g[2],
								"{DEPART}", g[3],
								"{LASTEDIT}", "")
							rawTables += raw.Replace(string(rawHTML))
						}
					}
				}

				var gamesList string
				res, err := db.Query("SELECT * FROM games")
				defer res.Close()
				if err != nil {
					log.Fatal(err)
				}

				for res.Next() {

					var names Games
					err := res.Scan(&names.id, &names.name, &names.image)
					if err != nil {
						log.Fatal(err)
					}
					gamesList += `<option>` + names.name + `</option> \n`
				}

				data, _ := ioutil.ReadFile("static/manage.html")
				raw := strings.NewReplacer("{USERNAME}", userName,
					"{MEMBERS}", strconv.Itoa(countRows("members")),
					"{ACTIVEMEMBERS}", strconv.Itoa(countSpecial("status", "Active")),
					"{MIAMEMBERS}", strconv.Itoa(countSpecial("status", "MIA")),
					"{BANNEDMEMBERS}", strconv.Itoa(countSpecial("status", "BANNED")),
					"{MEMUSERNAME}", username,
					"{MEMFORUMURL}", forumurl,
					"{MEMRANK}", rank,
					"{MEMSTATUS}", status,
					"{ROLLCALL}", rollcall,
					"{ADMINNOTES}", notes,
					"<!--{GAMESTABLE}-->", rawTables,
					"<!--{GAMENAMES}-->", gamesList,
					"{CNAME}", getSettings("name"),
					"{MEMID}", uid,
				)

				readyHTML := raw.Replace(string(data))
				fmt.Fprintf(w, readyHTML)
			} else {
				fmt.Println("Err: " + err.Error())
				fmt.Fprintf(w, "Error reading database... Try again later.")
			}
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func saveSettingsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		var comName = r.Form.Get("cname")
		fmt.Println(comName)
		_, err := db.Exec("UPDATE `settings` SET `data`='" + comName + "' WHERE setting='name'")
		if err == nil {
			fmt.Fprintf(w, "success")
		} else {
			fmt.Fprintf(w, "error")
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	userName := getUserName(r)
	if userName != "" {
		var notes string

		err = db.QueryRow("SELECT notes FROM admin where log = ?", 1).Scan(&notes)
		if err != nil {
			panic(err.Error()) 
		}

		data, _ := ioutil.ReadFile("static/main.html")
		dataLog, _ := ioutil.ReadFile("logs/agc-rp-admin.log")
		raw := strings.NewReplacer("{USERNAME}", userName,
			"{MEMBERS}", strconv.Itoa(countRows("members")),
			"{ACTIVEMEMBERS}", strconv.Itoa(countSpecial("status", "Active")),
			"{MIAMEMBERS}", strconv.Itoa(countSpecial("status", "MIA")),
			"{BANNEDMEMBERS}", strconv.Itoa(countSpecial("status", "BANNED")),
			"{LOGTEXT}", string(dataLog),
			"{NOTESTEXT}", string(notes),
			"{CNAME}", getSettings("name"),
		)

		readyHTML := raw.Replace(string(data))
		fmt.Fprintf(w, readyHTML)

	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func adminnotesHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		adminNotes := r.FormValue("notes")

		_, err := db.Exec("UPDATE `admin` SET `notes`='" + adminNotes + "' WHERE log='1'")

		if err == nil {
			AdminLog.Println(userName + " edited admin notes")
			fmt.Fprintf(w, "success")
		} else {
			fmt.Fprintf(w, "error")
		}
	}
}

func ipHandler(w http.ResponseWriter, r *http.Request) {

	ip := strings.Split(r.RemoteAddr, ":")[0]
	w.Write([]byte(ip))

}

func NotFound(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/404.html")
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	userName := getUserName(r)
	if userName != "" {
		r.ParseForm()
		TYPE := r.Form.Get("type")
		if TYPE == "member" {
			UID := r.Form.Get("uid")
			var tmpuid string
			err := db.QueryRow("SELECT uid FROM members WHERE uid=?", UID).Scan(&tmpuid)
			if err != sql.ErrNoRows {
				_ = db.QueryRow("DELETE FROM members WHERE uid=?", UID)
				AdminLog.Println(userName + " deleted a member from the database UID=" + UID)
			}
		} else if TYPE == "memgame" {
			memberID := r.Form.Get("user")
			game := r.Form.Get("game")


			var games string
			err := db.QueryRow("SELECT games FROM members WHERE uid=?", memberID).Scan(&games)
			if err != sql.ErrNoRows {
				s := strings.Split(games, ",")
				for i := 0; i < len(s); i++ {
					if strings.Contains(s[i], game) {
						var adjustedgames string
						adjustedgames = strings.Replace(games, s[i]+",", "", -1)
						_, err := db.Exec("UPDATE `members` SET `games`='" + adjustedgames + "' WHERE uid='" + memberID + "'")
						if err == nil {
							AdminLog.Println(userName + " deleted a member [ " + memberID + " ]'s game [ " + game + " ]from the database.")
							fmt.Fprintf(w, "success")
						} else {
							fmt.Println(err.Error())
							AdminLog.Println(userName + " tried to deleted a member [ " + memberID + " ]'s game [ " + game + " ]from the database.")
							fmt.Fprintf(w, "error")
						}
					}
				}
			}
		} else if TYPE == "game" {
			game := r.Form.Get("game")
			var tmpuid string
			err := db.QueryRow("SELECT name FROM games WHERE name=?", game).Scan(&tmpuid)
			if err != sql.ErrNoRows {
				_ = db.QueryRow("DELETE FROM games WHERE name=?", game)
				AdminLog.Println(userName + " deleted a game [ " + game + " ] from the database")
			}
		} else if TYPE == "admin" {
			user := r.Form.Get("user")
			var tmpuid string
			err := db.QueryRow("SELECT username FROM admins WHERE username=?", user).Scan(&tmpuid)
			if err != sql.ErrNoRows {
				_ = db.QueryRow("DELETE FROM admins WHERE username=?", user)
				AdminLog.Println(userName + " deleted a admin [ " + user + " ] from the database")
			}
		} else if TYPE == "server" {
			app := r.Form.Get("app")
			var tmpuid string
			err := db.QueryRow("SELECT application FROM servers WHERE application=?", app).Scan(&tmpuid)
			if err != sql.ErrNoRows {
				_ = db.QueryRow("DELETE FROM servers WHERE application=?", app)
				AdminLog.Println(userName + " deleted a server [ " + app + " ] from the database")
			}
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func addAdminHandler(w http.ResponseWriter, r *http.Request) {
	userName := getUserName(r)
	if userName != "" {
		r.ParseForm()
		username := r.Form.Get("user")
		password := r.Form.Get("pass")
		var saltedPass = md5Hash(md5Salt + "+" + password)
		_, err := db.Exec("INSERT INTO admins( username, password, lastseen, ip) VALUES( ?, ?, ?, ?)", username, saltedPass, "Never", "127.0.0.1")
		if err == nil {
			AdminLog.Println(userName + " added admin [ " + username + " ] to database.")
			fmt.Fprintf(w, "success")
		} else {
			fmt.Fprintf(w, "error")
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func adminAddGame(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(100000)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	m := r.MultipartForm
	files := m.File["myfiles"]
	gameImageName := m.Value["gamename"]
	for i, _ := range files {
		file, err := files[i].Open()
		defer file.Close()
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		dst, err := os.Create("./static/images/gameicons/" + gameImageName[0] + ".png")
		defer dst.Close()
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(dst, file); err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}

	_, err = db.Exec("INSERT INTO games( name, image) VALUES( ?, ?)", gameImageName[0], `..\static\images\gameicons\`+gameImageName[0]+`.png`)
	if err == nil {
		AdminLog.Println("Admin added new game [ " + gameImageName[0] + " ] to database.")
		redirectTarget := "/settings?alert=yes"
		http.Redirect(w, r, redirectTarget, 302)
	} else {
		redirectTarget := "/settings?alert=error"
		http.Redirect(w, r, redirectTarget, 302)
	}

}

func addMemberGameHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		mid := r.Form.Get("uid")
		game := r.Form.Get("selectedgame")
		username := r.Form.Get("username")
		rank := r.Form.Get("rank")
		department := r.Form.Get("depart")
		var memberGames string
		err = db.QueryRow("SELECT `games`  FROM `members` where `username` = ?", mid).Scan(&memberGames)
		if err != nil {
			fmt.Fprintf(w, "error")
		} else {
			memberGames += game + "|" + username + "|" + rank + "|" + department + "|" + time.Now().Format(time.RFC822) + ","
			_, err = db.Exec("UPDATE `members` SET `games`='" + memberGames + "' WHERE username='" + mid + "'")
			if err == nil {
				AdminLog.Println(userName + " added new game [ " + game + " ] to member [ " + mid + " ]'s info.")

				fmt.Fprintf(w, "success")
			} else {
				fmt.Fprintf(w, "error")
			}
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func editMemberHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		mUID := r.Form.Get("uid")
		mUser := r.Form.Get("username")
		mURL := r.Form.Get("forumurl")
		mRank := r.Form.Get("ranks")
		mStatus := r.Form.Get("status")
		mRollCall := r.Form.Get("rollcall")
		mNotes := r.Form.Get("notes")


		_, err = db.Exec("UPDATE `members` SET `username`='" + mUser + "', `forumurl`='" + mURL + "', `rank`='" + mRank + "', `status`='" + mStatus + "', `rollcall`='" + mRollCall + "', `notes`='" + mNotes + "' WHERE uid='" + mUID + "'")
		if err == nil {

			AdminLog.Println(userName + " edited member [ " + mUser + " ]'s info.")
			fmt.Fprintf(w, "success")
		} else {
			fmt.Println("ERR2: " + err.Error())
			fmt.Fprintf(w, "error")
		}

	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func addServerHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		r.ParseForm()
		sName := r.Form.Get("sName")
		sApp := r.Form.Get("sApp")
		sLoc := r.Form.Get("sLoc")
		_, err := db.Exec("INSERT INTO servers( name, application, location, state) VALUES( ?, ?, ?, ?)", sName, sApp, sLoc, "STOPPED")
		if err == nil {
			AdminLog.Println(userName + " added server [ " + sApp + " ] to database.")
			fmt.Fprintf(w, "success")
		} else {
			fmt.Println(err.Error())
			fmt.Fprintf(w, "error")
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func serverHandler(w http.ResponseWriter, r *http.Request) {
	userName := getUserName(r)
	if userName != "" {
		var sUID, sName, sApp, sLoc, sState string
		var rawTables string = ""
		rows, _ := db.Query("SELECT uid, name, application, location, state FROM servers")

		for rows.Next() {
			_ = rows.Scan(&sUID, &sName, &sApp, &sLoc, &sState)
			var state string = ""
			var stateIco string = ""
			if sState == "RUNNING" {
				state = `<span style="color: #00ff00;">RUNNING</span>`
				stateIco = "fa-stop"
			} else if sState == "STOPPED" {
				state = `<span style="color: #ff0000;">STOPPED</span>`
				stateIco = "fa-play"
			}

			rawHTML, _ := ioutil.ReadFile("static/serverTable.html")
			raw := strings.NewReplacer("{NAME}", sName,
				"{Application}", sApp,
				"{PATH}", sLoc,
				"{STATUS}", state,
				"{STATEICON}", stateIco,
				"{UID}", sUID,
			)
			rawTables += raw.Replace(string(rawHTML))
		}

		data, _ := ioutil.ReadFile("static/servers.html")

		raw := strings.NewReplacer("{USERNAME}", userName,
			"{MEMBERS}", strconv.Itoa(countRows("members")),
			"{ACTIVEMEMBERS}", strconv.Itoa(countSpecial("status", "Active")),
			"{MIAMEMBERS}", strconv.Itoa(countSpecial("status", "MIA")),
			"{BANNEDMEMBERS}", strconv.Itoa(countSpecial("status", "BANNED")),
			"{CNAME}", getSettings("name"),
			"<!--ServersTable-->", rawTables,
		)

		readyHTML := raw.Replace(string(data))
		fmt.Fprintf(w, readyHTML)
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func countRows(table string) int { 
	rows, _ := db.Query("SELECT COUNT(*) AS count FROM " + table + "")
	var count int
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&count)
	}
	return count
}

func countSpecial(Col string, val string) int {
	rows, _ := db.Query("SELECT COUNT(*) AS count FROM members WHERE " + Col + " = '" + val + "'")
	var count int
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&count)
	}
	return count
}

func editMemberGameInfoHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		mUID := r.Form.Get("uid")
		gName := r.Form.Get("gName")
		gUser := r.Form.Get("gUser")
		gRank := r.Form.Get("gRank")
		gDepart := r.Form.Get("gDepart")

		var memberGames string
		err = db.QueryRow("SELECT `games`  FROM `members` where `uid` = ?", mUID).Scan(&memberGames)
		if err != nil {
			fmt.Fprintf(w, "error")
		} else {
			s := strings.Split(memberGames, ",")
			for i := 0; i < len(s); i++ {
				if strings.Contains(s[i], "|") {
					if strings.Contains(s[i], gName) {
						if !strings.Contains(s[i], gName+"|"+gUser+"|"+gRank+"|"+gDepart+"|") {
							memberGames = strings.Replace(memberGames, s[i], gName+"|"+gUser+"|"+gRank+"|"+gDepart+"|"+time.Now().Format(time.RFC822), -1)
						}
					}
				}
			}
			_, err = db.Exec("UPDATE `members` SET `games`='" + memberGames + "' WHERE uid='" + mUID + "'")
			if err == nil {
				fmt.Fprintf(w, "success")
			} else {
				fmt.Fprintf(w, "error")
			}
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func editServerInfoHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		sUID := r.Form.Get("sUID")
		sName := r.Form.Get("sName")
		sApp := r.Form.Get("sApp")
		sLoc := r.Form.Get("sLoc")
		var dbSUID string
		err = db.QueryRow("SELECT `name`  FROM `servers` where `uid` = ?", sUID).Scan(&dbSUID)
		if err != nil {
			fmt.Fprintf(w, "error")
		} else {
			var fixed string
			fixed = strings.ReplaceAll(sLoc, `\`, `\\`)
			_, err = db.Exec("UPDATE `servers` SET `name`='" + sName + "', `application`='" + sApp + "', `location`='" + fixed + "' WHERE uid='" + sUID + "'")
			if err == nil {
				fmt.Fprintf(w, "success")
			} else {
				fmt.Fprintf(w, "error")
			}
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func FormTest(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	for key, values := range r.Form {
		for _, value := range values {
			fmt.Println(key, value)
		}
	}
	fmt.Fprintf(w, "success")
}

func faviconHandle(response http.ResponseWriter, request *http.Request) {
	http.ServeFile(response, request, "static/images/favicon.ico")
}

func runningProcessCheck(proc string) bool {
	var dst []Win32_Process
	q := wmi.CreateQuery(&dst, "")
	err := wmi.Query(q, &dst)
	if err != nil {
		return false
	}
	for _, v := range dst {
		if bytes.Contains([]byte(strings.ToLower(v.Name)), []byte(strings.ToLower(proc))) {
			return true
		}
	}
	return false
}

func handleProgramState(mode int, application string, location string) {
	if mode == 0 { //Start
		CMD := exec.Command("cmd", "/Q", "/C", "start "+location)
		CMD.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		CMD.Start()
	} else if mode == 1 { //Stop
		CMD := exec.Command("cmd", "/Q", "/C", "taskkill /F /IM "+application+".exe")
		CMD.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		CMD.Start()
	} else if mode == 2 { //Restart
		CMD := exec.Command("cmd", "/Q", "/C", "taskkill /F /IM "+application+".exe")
		CMD.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		CMD.Start()
		time.Sleep(2 * time.Second)
		CMD = exec.Command("cmd", "/Q", "/C", "start "+location)
		CMD.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		CMD.Start()
	}
}

func startStopServerHandle(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := getUserName(r)
	if userName != "" {
		mode := r.Form.Get("mode")
		application := r.Form.Get("app")

		var sName, sApplication, sLocation, sState string
		err = db.QueryRow("SELECT `name`, `application`, `location`, `state`  FROM `servers` where `uid` = ?", application).Scan(&sName, &sApplication, &sLocation, &sState)
		if err != nil {

		} else {

			if mode == "0" {
				if sState == "RUNNING" {
					go handleProgramState(1, sApplication, sLocation)
					fmt.Fprintf(w, "success")
				} else if sState == "STOPPED" {
					go handleProgramState(0, sApplication, sLocation)
					fmt.Fprintf(w, "success")
				}
			} else if mode == "1" { 
				go handleProgramState(2, sApplication, sLocation)
				fmt.Fprintf(w, "success")
			}
		}
	} else {
		data, _ := ioutil.ReadFile("static/login.html")
		var fixedhtml = string(data)
		fixedhtml = strings.Replace(string(data), "{CNAME}", getSettings("name"), -1)
		fmt.Fprintf(w, fixedhtml)
	}
}

func feedHandle(w http.ResponseWriter, r *http.Request) {

}

func goServer() {
	Log.Println("Router Started")

	r := mux.NewRouter()

	r.HandleFunc("/", dashboardHandler)
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/logout", logoutHandler)

	r.HandleFunc("/roster", rosterHandler)
	r.HandleFunc("/addmember", addMemberHandler).Methods("POST")
	r.HandleFunc("/manage", manageHandler)
	r.HandleFunc("/editgame", addMemberGameHandler)
	r.HandleFunc("/editmember", editMemberHandler)
	r.HandleFunc("/settings", settingsHandler)
	r.HandleFunc("/savesettings", saveSettingsHandler)
	r.HandleFunc("/server", serverHandler)
	r.HandleFunc("/addserver", addServerHandler)
	r.HandleFunc("/addadmin", addAdminHandler)
	r.HandleFunc("/rollcall", rollcallHandler)
	r.HandleFunc("/eginfo", editMemberGameInfoHandler)
	r.HandleFunc("/esinfo", editServerInfoHandler)
	r.HandleFunc("/sserver", startStopServerHandle)
	r.HandleFunc("/upload", adminAddGame).Methods("POST")

	r.HandleFunc("/adminnotes", adminnotesHandler).Methods("POST")

	r.HandleFunc("/ip", ipHandler)
	r.HandleFunc("/delete", deleteHandler)

	r.HandleFunc("/favicon.ico", faviconHandle)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	r.NotFoundHandler = http.HandlerFunc(NotFound)

	srv := &http.Server{
		Handler: r,
		Addr:    ":" + serverPort,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	log.Fatal(srv.ListenAndServe())

}

func daemon() {
	for {
		var sApp string
		rows, err := db.Query("SELECT application FROM servers")
		if err == nil {
		} else {
		}
		for rows.Next() {
			_ = rows.Scan(&sApp)

			var state string = ""
			if runningProcessCheck(sApp) {
				state = `RUNNING`
			} else {
				state = `STOPPED`
			}
			_, err = db.Exec("UPDATE `servers` SET `state`='" + state + "' WHERE application='" + sApp + "'")
			if err == nil {
			} else {
			}
		}
		time.Sleep(15 * time.Second)
	}
}

func init() {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		Log.Println("Goodnight.")
		os.Exit(1)
	}()

	NewLog("logs/main.log")
	NewAdminLog("logs/admin.log")
}

func main() {
	Log.Println("Control Server started")
	Log.Println("Control Server running on port " + serverPort)

	fmt.Println("Control v1.02")
	fmt.Println("https://github.com/SaturnsVoid")


	Log.Println("Connecting to MySQL server...")
	db, err = sql.Open("mysql", mySQLUsername+":"+mySQLPassword+"@tcp("+mySQLHost+")/"+mySQLDatabase)
	if err != nil {
		Log.Fatalln("[!] CHECK MYSQL SETTINGS! [!]")
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		Log.Fatalln("[!] CHECK IF MYSQL SERVER IS ONLINE! [!]")
	}
	Log.Println("Connected to MySQL server.")
	Log.Println("Starting HTML server...")
	go daemon()
	go goServer()

	for {
		time.Sleep(1 * time.Second)
	}
}
