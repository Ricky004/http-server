package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
)


const (
	PROTOCOL              = "HTTP/1.1 "
	HTTP_STATUS_OK        = "200 OK"
	HTTP_STATUS_NOT_FOUND = "404 Not Found"
	HTTP_STATUS_CREATED   = "201 Created"
	INTERNAL_SERVER_ERROR = "500 Internal Server Error"
	CONTENT_LENGTH        = "Content-Length: "
	CONTENT_TEXT          = "Content-Type: text/plain"
	CONTENT_OCTET         = "Content-Type: application/octet-stream"
)

var dir string

func main() {
	flag.StringVar(&dir, "directory", "", "Directory to server")
	flag.Parse()

	l, err := net.Listen("tcp", ":4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	fmt.Println("Server listening on port 4221")

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handelConnection(conn)
	}
}

func handelConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 4096*4)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}

	request := ParseRequest(buf[:n])

	isEcho := strings.Contains(request.Path, "/echo/")
	isFile := strings.Contains(request.Path, "/files/")

	if request.Path == "/user-agent" || request.Path == "/" {
		response := PROTOCOL + HTTP_STATUS_OK + "\r\n" + CONTENT_TEXT + "\r\n" + CONTENT_LENGTH + fmt.Sprint(len(request.UserAgent)) + "\r\n\r\n" + request.UserAgent + "\r\n"
		conn.Write([]byte(response))

	} else if isEcho {
		responseBody := strings.Split(request.Path, "/echo/")
		response := PROTOCOL + HTTP_STATUS_OK + "\r\n" + CONTENT_TEXT + "\r\n" + CONTENT_LENGTH + fmt.Sprintf("%d", len(responseBody[1])) + "\r\n\r\n" + responseBody[1] + "\r\n"
		conn.Write([]byte(response))

	} else if isFile {
		if request.Method == "GET" {
			fileName := strings.Split(request.Path, "/files/")

			if len(fileName) != 2 {
				response := PROTOCOL + HTTP_STATUS_NOT_FOUND + "\r\n" + CONTENT_LENGTH + "\r\n\r\n"
				conn.Write([]byte(response))
				return
			}

			content, err := readFileContent(fileName[1])
			if err != nil {
				response := PROTOCOL + HTTP_STATUS_NOT_FOUND + "\r\n" + CONTENT_LENGTH + "\r\n\r\n"
				conn.Write([]byte(response))
				return
			}

			response := PROTOCOL + HTTP_STATUS_OK + "\r\n" + CONTENT_OCTET + "\r\n" + CONTENT_LENGTH + fmt.Sprintf("%d", len(content)) + "\r\n\r\n" + string(content) + "\r\n"
			conn.Write([]byte(response))
		} else if request.Method == "POST" {
			fileName := strings.TrimPrefix(request.Path, "/files/")
			savefile(conn, fileName, buf[:n])
		}

	} else {
		response := PROTOCOL + HTTP_STATUS_NOT_FOUND + "\r\n" + CONTENT_LENGTH + "\r\n\r\n"
		conn.Write([]byte(response))
	}

}

func readFileContent(fileName string) (string, error) {
	filePath := path.Join(dir, fileName)

	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func savefile(conn net.Conn, fileName string, requestBytes []byte) {
	content := strings.TrimSpace(string(requestBytes))

	parts := strings.SplitN(content, "\r\n\r\n", 2)
	if len(parts) != 2 {
		response := PROTOCOL + INTERNAL_SERVER_ERROR + "\r\n" + CONTENT_LENGTH + "\r\n\r\n"
		conn.Write([]byte(response))
		return
	}

	body := parts[1]

	path := dir + string(os.PathSeparator) + fileName

	err := os.WriteFile(path, []byte(body), 0666)
	if err != nil {
		response := PROTOCOL + INTERNAL_SERVER_ERROR + "\r\n" + CONTENT_LENGTH + "\r\n\r\n"
		conn.Write([]byte(response))
		return
	}
	
	response := PROTOCOL + HTTP_STATUS_CREATED + "\r\n" + CONTENT_OCTET + "\r\n" + CONTENT_LENGTH + fmt.Sprintf("%d", len(requestBytes)) + "\r\n\r\n" + string(requestBytes) + "\r\n"
	conn.Write([]byte(response))
}

type HttpRequest struct {
	Method    string
	Path      string
	Ver       string
	UserAgent string
	Host      string
}

func ParseRequest(b []byte) *HttpRequest {
	var req HttpRequest
	lines := strings.Split(string(b), "\r\n")
	for idx, line := range lines {
		parts := strings.Split(line, " ")

		switch idx {
		case 0:
			if len(parts) != 3 {
				continue
			}
			req.Method = parts[0]
			req.Path = parts[1]
			req.Ver = parts[2]
		case 1:
			if len(parts) != 2 {
				continue
			}
			req.Host = parts[1]
		case 2:
			if len(parts) != 2 {
				continue
			}
			req.UserAgent = parts[1]
		}
	}

	return &req
}
