package iot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"
	"vrchat_osc/module/config"

	"github.com/gorilla/mux"
	"github.com/hypebeast/go-osc/osc"
)

var (
	OffComputerTimeout  bool
	OffComputerTimeouts int64

	ContinueNextTasks bool
)

func Init() {
	x := mux.NewRouter()
	x.HandleFunc("/off_computer", handle_off_computer)
	x.HandleFunc("/stop_off_computer", handle_stop_off_computer)
	x.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./vrchat_osc.log")
	})

	http.ListenAndServe(":"+fmt.Sprint(config.IotPort), loggingMiddleware(x))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("IOT %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func handle_stop_off_computer(w http.ResponseWriter, r *http.Request) {
	if OffComputerTimeout {
		OffComputerTimeout = false
		exec.Command("cmd", "/c", "shutdown", "/a").Run()
		client := osc.NewClient("localhost", 9000)
		msg := osc.NewMessage("/chatbox/input")
		msg.Append("- 远程关机被打断 -")
		msg.Append(true)
		msg.Append(false)
		client.Send(msg)
		result := map[string]any{
			"code":    0,
			"message": "成功",
		}
		data, _ := json.Marshal(result)
		w.Write(data)
	} else {
		ContinueNextTasks = !ContinueNextTasks
		if ContinueNextTasks {
			result := map[string]any{
				"code":    1,
				"message": "将跳过下一次关机指令",
			}
			data, _ := json.Marshal(result)
			w.Write(data)
		} else {
			result := map[string]any{
				"code":    2,
				"message": "已去除跳过",
			}
			data, _ := json.Marshal(result)
			w.Write(data)
		}

	}
}

func handle_off_computer(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Has("message") {
		if ContinueNextTasks {
			result := map[string]any{
				"code":    -1,
				"message": "任务被跳过",
			}
			data, _ := json.Marshal(result)
			w.Write(data)
			ContinueNextTasks = false
			return
		}
		result := map[string]any{
			"code":    0,
			"message": "等待主人确认",
		}
		data, _ := json.Marshal(result)
		w.Write(data)
		OffComputer(r.URL.Query().Get("message"))
	} else {
		result := map[string]any{
			"code":    1,
			"message": "失败",
		}
		data, _ := json.Marshal(result)
		w.Write(data)
	}
}

func OffComputer(message string) {
	OffComputerTimeout = true
	OffComputerTimeouts = time.Now().Unix()
	config.KeyboardInputTime = time.Now().Unix()
	client := osc.NewClient("localhost", 9000)
	msg := osc.NewMessage("/chatbox/input")
	msg.Append("- 远程关机 -\n设备被远程操控或定时任务关机，被打断则无效")
	msg.Append(true)
	msg.Append(false)
	client.Send(msg)
	exec.Command("cmd", "/c", "shutdown", "/s", "/t", "120", "/c", message).Run()
}
