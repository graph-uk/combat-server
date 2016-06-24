package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (t *CombatServer) createSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionName := strconv.FormatInt(time.Now().UnixNano(), 10)

	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("uploadfile")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	sessionParams := r.FormValue("params")
	if sessionParams == "" {
		fmt.Println("cannot extract session params")
		return
	}

	os.MkdirAll("./sessions/"+sessionName, 0777)
	f, err := os.OpenFile("./sessions/"+sessionName+"/archived.zip", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	req, err := t.mdb.DB.Prepare("INSERT INTO Sessions(id,params) VALUES(?,?)")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	_, err = req.Exec(sessionName, sessionParams)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	io.WriteString(w, sessionName)
	fmt.Println(r.Host + " Create new session: " + sessionName + " " + sessionParams)

	go t.doCasesExplore(sessionParams, sessionName)
}

func (t *CombatServer) doCasesExplore(params, sessionID string) error {
	//fmt.Println("CasesExplore")
	err := unzip("./sessions/"+sessionID+"/archived.zip", "./sessions/"+sessionID+"/unarch")
	if err != nil {
		fmt.Print(err.Error())
		return err
	}
	os.Chdir("./sessions/" + sessionID + "/unarch/src/Tests")
	rootTestsPath, _ := os.Getwd()
	rootTestsPath += string(os.PathSeparator) + ".." + string(os.PathSeparator) + ".."

	command := []string{"cases"}
	for _, curParameter := range strings.Split(params, " ") {
		//fmt.Println("#" + curElement)
		if strings.TrimSpace(curParameter) != "" {
			command = append(command, curParameter)
		}
	}
	//fmt.Println(command)
	cmd := exec.Command("combat", command...)
	cmd.Env = t.addToGOPath(rootTestsPath)

	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	//fmt.Println(command)
	//fmt.Print("Run combat cases... ")
	exitStatus := cmd.Run()

	os.Chdir(t.startPath)
	if exitStatus == nil {
		//fmt.Println("Ok")
		//t.postCases(out.String(), sessionID)
		t.setCasesForSession(out.String(), sessionID)
		//fmt.Println(out.String())
	} else {
		fmt.Println("Cannot extract cases")
		fmt.Println(out.String())
		fmt.Println(outErr.String())
		return errors.New("Cannot extract combat cases.")
	}

	return nil
}

func (t *CombatServer) setCasesForSession(sessionCases, sessionID string) error {
	sessionCasesArr := strings.Split(sessionCases, "\n")

	req, err := t.mdb.DB.Prepare("INSERT INTO Cases(cmdline, sessionID) VALUES(?,?)")
	if err != nil {
		fmt.Println(err)
		return (err)
	}

	casesCount := 0
	for _, curCase := range sessionCasesArr {
		curCaseCleared := strings.TrimSpace(curCase)
		if curCaseCleared != "" {
			casesCount++
			_, err = req.Exec(curCase, sessionID)
			if err != nil {
				fmt.Println(err)
				return (err)
			}
		}
	}

	req, err = t.mdb.DB.Prepare("UPDATE Sessions SET status=? WHERE id=?")
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = req.Exec(1, sessionID) // cases ExploringStarted
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("Explored " + strconv.Itoa(casesCount) + " cases for session: " + sessionID)
	return nil
}