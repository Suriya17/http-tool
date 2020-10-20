package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// Parses URL and returns host and resource string
func parseURL(url string) (host, resource string) {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		url = strings.Split(url, "//")[1]
	}

	splitURL := strings.SplitN(url, "/", 2)

	host = splitURL[0]

	resource = "/"

	if len(splitURL) > 1 {
		resource += splitURL[1]
	}

	return host, resource
}

// Parses raw HTTP response and return the HTTP response code and Message-body
func parseHTTPResponse(buf *bytes.Buffer) (int, string) {
	s, err := buf.ReadString('\n')

	if err != nil {
		fmt.Errorf("Error parsing Response")
		return 400, ""
	}

	line := strings.Split(s, " ")

	returnCode, err := strconv.Atoi(line[1])

	if err != nil {
		fmt.Errorf("Error Response code")
		return 400, ""
	}

	content := buf.Bytes()

	var flag bool = false

	for i := 1; i < len(content); i++ {
		if content[i-1] == '\r' && content[i] == '\n' {
			if flag == false {
				flag = true
				i++
			} else {
				body := string(content[i+1:])
				return returnCode, body
			}
		} else {
			flag = false
		}
	}

	return returnCode, ""
}

// Sends a single HTTP request to url. Returns response code and message-body
func sendHTTPReq(url string) (int, string) {
	h, r := parseURL(url)

	conn, err := net.Dial("tcp", h+":80")
	defer conn.Close()

	if err != nil {
		fmt.Errorf("Invalid URL")
		return 400, ""
	}

	conn.Write([]byte("GET " + r + " HTTP/1.0\r\nHost: " + h + "\r\n\r\n"))

	var buf bytes.Buffer

	io.Copy(&buf, conn)

	return parseHTTPResponse(&buf)
}

// Sends a single HTTPS request to url. Returns response code and message-body
func sendHTTPSReq(url string) (int, string) {
	h, r := parseURL(url)

	conn, err := tls.Dial("tcp", h+":443", nil)
	defer conn.Close()

	if err != nil {
		fmt.Errorf("Invalid URL")
		return 400, ""
	}

	conn.Write([]byte("GET " + r + " HTTP/1.0\r\nHost: " + h + "\r\n\r\n"))

	var buf bytes.Buffer

	io.Copy(&buf, conn)

	return parseHTTPResponse(&buf)
}

// Sends and prints the output of a HTTP request
func sendAndPrintHTTP(url string) {
	returnCode, resp := sendHTTPReq(url)

	if returnCode != 200 {
		fmt.Println("Error! HTTP return code : ", returnCode)
		return
	}
	fmt.Print(resp)
}

// Sends and prints the output of a HTTPS request
func sendAndPrintHTTPS(url string) {
	returnCode, resp := sendHTTPSReq(url)

	if returnCode != 200 {
		fmt.Println("Error! HTTP return code : ", returnCode)
		return
	}
	fmt.Print(resp)
}

// Send multiple request for profiling
func sendMultipleReqs(url string, httpsFlag bool, numReqs int) {
	responseTimes := make([]int, 0)
	errorCodes := make([]int, 0)
	var largestResponseSize int = 0
	var smallestResponseSize int = 100000
	var successes int = 0
	var responseTimesSum float64 = 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	if httpsFlag {
		for i := 0; i < numReqs; i++ {
			wg.Add(1)
			go func() {
				start := time.Now()
				errorCode, resp := sendHTTPSReq(url)
				duration := time.Since(start)

				mu.Lock()
				if errorCode != 200 {
					errorCodes = append(errorCodes, errorCode)
					mu.Unlock()
					wg.Done()
					return
				}
				successes++
				responseTimesSum += float64(duration.Milliseconds())
				largestResponseSize = Max(largestResponseSize, len(resp))
				smallestResponseSize = Min(smallestResponseSize, len(resp))
				responseTimes = append(responseTimes, int(duration.Milliseconds()))
				mu.Unlock()
				wg.Done()
			}()
		}
	} else {
		for i := 0; i < numReqs; i++ {
			wg.Add(1)
			go func() {
				start := time.Now()
				errorCode, resp := sendHTTPReq(url)
				duration := time.Since(start)

				mu.Lock()
				if errorCode != 200 {
					errorCodes = append(errorCodes, errorCode)
					mu.Unlock()
					wg.Done()
					return
				}
				successes++
				responseTimesSum += float64(duration.Milliseconds())
				largestResponseSize = Max(largestResponseSize, len(resp))
				smallestResponseSize = Min(smallestResponseSize, len(resp))
				responseTimes = append(responseTimes, int(duration.Milliseconds()))
				mu.Unlock()
				wg.Done()
			}()
		}
	}
	wg.Wait()
	sort.Ints(responseTimes)
	fmt.Println("URL : ", url)
	fmt.Println("Num of requests : ", numReqs)
	fmt.Println("Num of successes : ", successes)
	if successes > 0 {
		fmt.Println("Fastest Response time : ", responseTimes[0])
		fmt.Println("Slowest Response time : ", responseTimes[len(responseTimes)-1])
		fmt.Println("Mean Response time : ", responseTimesSum/float64(successes))
		fmt.Println("Median Response time : ", (responseTimes[len(responseTimes)/2]+responseTimes[(len(responseTimes)-1)/2])/2)
		fmt.Println("Non-Success error codes : ", errorCodes)
		fmt.Println("Largest Response size(Only message-body considered) : ", largestResponseSize)
		fmt.Println("Smallest Response size(Only message-body considered) : ", smallestResponseSize)
	}
}

func main() {
	var https bool
	var multipleReqs int
	var url string

	flag.BoolVar(&https, "https", false, "HTTPS is off by default. Set this for HTTPS")
	flag.IntVar(&multipleReqs, "profile", 0, "Set number of reqs here. Default value of 0 means print output mode")
	flag.StringVar(&url, "url", "", "Specify URL")

	flag.Parse()

	if len(url) == 0 {
		fmt.Println("URL not specified!")
		return
	}

	if multipleReqs > 0 {
		sendMultipleReqs(url, https, multipleReqs)
	} else {
		if https {
			sendAndPrintHTTPS(url)
		} else {
			sendAndPrintHTTP(url)
		}
	}

}
