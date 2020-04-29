package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
	"launchpad.net/gnuflag"
)

const (
	baseURL = "https://ynn-store.herokuapp.com"
)

/*
Usage ...
*/
func Usage() {
	executable := filepath.Base(os.Args[0])
	fmt.Println()
	fmt.Println(executable, "up <local file/dir name> <namespace> [remote file name]")
	fmt.Println(executable, "down <namespace> <remote file name> [local file to save to]")
}

func up(password bool) {
	if gnuflag.NArg() < 3 {
		Usage()
		return
	}

	namespace := gnuflag.Arg(1)
	localPath := gnuflag.Arg(2)
	var remotePath string = path.Base(localPath)
	if gnuflag.NArg() > 3 {
		remotePath = path.Base(gnuflag.Arg(3))
	}

	url := baseURL + path.Join("/files", namespace, remotePath)
	file, err := os.Open(localPath)
	if err != nil {
		fmt.Println("Error while opening file: ", err.Error())
		return
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", remotePath)
	if err != nil {
		fmt.Println("Error while creating a multipart file: ", err.Error())
		return
	}

	pass := ""
	if password {
		fmt.Print("Password: ")
		passBytes, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Println("Error reading password", err.Error())
			return
		}
		pass = string(passBytes)
		if pass == "" {
			fmt.Println("Error: empty password is not perimtted")
			return
		}
		fmt.Println()
	}

	_, err = io.Copy(part, file)
	err = writer.Close()
	if err != nil {
		fmt.Println("Error while closing writer: ", err.Error())
		return
	}

	req, err := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if pass != "" {
		req.Header.Set("Authorization", pass)
	}
	if err != nil {
		fmt.Println("Error while creating request: ", err.Error())
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error while sending request: ", err.Error())
		return
	}

	body = &bytes.Buffer{}
	defer resp.Body.Close()
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		fmt.Println("Error reading data from response: ", err.Error())
		return
	}

	fmt.Println(body)
}

func down(password bool) {
	if gnuflag.NArg() < 3 {
		Usage()
		return
	}

	namespace := gnuflag.Arg(1)
	remotePath := gnuflag.Arg(2)
	localPath := path.Base(remotePath)
	if gnuflag.NArg() > 3 {
		localPath = gnuflag.Arg(3)
	}

	pass := ""
	if password {
		fmt.Print("Password: ")
		passBytes, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Println("Error reading password: ", err.Error())
			return
		}
		pass = string(passBytes)
		fmt.Println()
	}

	file, err := os.Create(localPath)
	if err != nil {
		fmt.Println("Error opening a file: ", err.Error())
	}
	defer file.Close()

	url := baseURL + path.Join("/files", namespace, remotePath)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating HTTP request: ", err.Error())
		return
	}
	if pass != "" {
		req.Header.Set("Authorization", pass)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making HTTP request: ", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		if !password {
			down(true)
		} else {
			fmt.Println("Bad password.")
		}
		return
	}

	n, err := io.Copy(file, resp.Body)
	if err != nil {
		fmt.Println("Error downloading file: ", err.Error())
		return
	}

	fmt.Println("Downloaded ", n, "bytes of data")
}

func main() {
	passwordL := gnuflag.Bool("password", false, "Provide this gnuflag and you'll be prompted for password")
	passwordS := gnuflag.Bool("p", false, "Same as --password")
	gnuflag.Parse(true)

	password := *passwordL || *passwordS

	if gnuflag.NArg() < 1 {
		gnuflag.Usage()
		Usage()
		return
	}

	verb := gnuflag.Arg(0)

	switch verb {
	case "up":
		up(password)
	case "down":
		down(password)
	default:
		Usage()
	}
}
