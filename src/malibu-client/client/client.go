package malibuClient

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"time"
)

type MalibuClient struct {
	serverURL string
	//sessionID             string
	sessionBeginTimestamp time.Time
	lastSTDOutMessage     string
	SessionTimeout        time.Duration
}

func (t *MalibuClient) getServerUrlFromCLI() (string, error) {
	if len(os.Args) < 2 {
		return "", errors.New("Server URL is required")
	}
	return os.Args[1], nil
}

func (t *MalibuClient) getSessionTimeoutFromCLI() (time.Duration, error) {
	mins, err := strconv.Atoi(os.Args[3])
	if err != nil {
		return time.Millisecond, errors.New(`Cannot parse session timeout. It should be int.`)
	}
	result := time.Duration(mins) * time.Minute
	return result, nil
}

func (t *MalibuClient) getTestsFolder() string {
	if len(os.Args) < 3 {
		return "./../.."
	}
	return os.Args[2]
}

func NewMalibuClient() (*MalibuClient, error) {
	var result MalibuClient
	var err error

	result.serverURL, err = result.getServerUrlFromCLI()
	if err != nil {
		return &result, err
	}

	result.SessionTimeout, err = result.getSessionTimeoutFromCLI()
	if err != nil {
		return &result, err
	}

	result.lastSTDOutMessage = ""
	return &result, nil
}

func (t *MalibuClient) packTests() (string, error) {
	fmt.Println("Packing tests")
	tmpFile, err := ioutil.TempFile("", "malibuSession")
	if err != nil {
		panic(err)
	}
	tmpFile.Close()
	zipit(t.getTestsFolder(), tmpFile.Name())
	return tmpFile.Name(), nil
}

func (t *MalibuClient) cleanupTests() error {
	fmt.Println("Cleanup tests")
	return nil
}

func (t *MalibuClient) getParams() string {
	params := ""
	for curArgIndex, curArg := range os.Args {
		if curArgIndex > 2 {
			params += curArg + " "
		}
	}
	return params
}

func (t *MalibuClient) createSessionOnServer(archiveFileName string) string {
	fmt.Print("Uploading session")
	sessionName := ""

	sessionName, err := postSession(archiveFileName, t.getParams(), t.serverURL+"/api/v1/sessions")
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}

	return sessionName
}

// CreateNewSession ...
func (t *MalibuClient) CreateNewSession() (string, error) {
	t.sessionBeginTimestamp = time.Now()
	err := t.cleanupTests()
	if err != nil {
		fmt.Println("Cannot cleanup tests")
		return "", err
	}

	testsArchiveFileName, err := t.packTests()
	if err != nil {
		fmt.Println("Cannot pack tests to zip archive")
		return "", err
	}

	sessionName := t.createSessionOnServer(testsArchiveFileName)
	if sessionName != "" {
		fmt.Println("Session status: " + t.serverURL + "/sessions/" + sessionName)
		return sessionName, nil
	}

	return "", nil
}

func (t *MalibuClient) GetSessionResult(sessionID string) (int, string) {
	countOfErrors := 1
	sessionError := `Unexpected session error`
	for {
		sessionStatusJSON, err := t.getSessionStatusJSON(sessionID)
		if err != nil {
			fmt.Println(err.Error())
		}
		//		fmt.Println(`*******************************` + sessionStatusJSON)
		var finished bool
		finished, countOfErrors, err, sessionError = t.printSessionStatusByJSON(sessionStatusJSON)
		//		fmt.Println(`*******************************`, finished, countOfErrors, err)
		if err == nil {
			if finished {
				break
			}
		}

		if time.Since(t.sessionBeginTimestamp) > t.SessionTimeout {
			fmt.Println(``)
			fmt.Println(`Timeout was reached, but session is still not finished. Check workers and start new session.`)
			os.Exit(1)
		}
		time.Sleep(5 * time.Second)
	}

	// cut microseconds
	timeLongStr := time.Since(t.sessionBeginTimestamp).String()
	r := regexp.MustCompile(`\.\d*s$`)
	timeShortStr := r.ReplaceAllString(timeLongStr, "s")

	fmt.Println("\r\nTime of testing: " + timeShortStr)
	if sessionError != `` {
		fmt.Println("Session error:", sessionError)
	}
	return countOfErrors, sessionError
}
