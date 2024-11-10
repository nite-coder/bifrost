package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

func main() {
	originalURL := "https://echo.free.beeceptor.com/spot/orders"

	// 解析 URL
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		fmt.Printf("URL parsing error: %v\n", err)
		return
	}

	// DNS lookup
	ips, err := net.LookupIP(parsedURL.Hostname())
	if err != nil {
		fmt.Printf("DNS lookup error: %v\n", err)
		return
	}

	if len(ips) == 0 {
		fmt.Println("No IP addresses found")
		return
	}

	// 使用第一個找到的 IP 建立新的 URL
	ipURL := fmt.Sprintf("https://%s%s", ips[0].String(), parsedURL.Path)

	// 建立 HTTP client
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName:         "echo.free.beeceptor.com", // magic part
				InsecureSkipVerify: true,
			},
		},
	}

	// 建立請求，使用 ipURL
	req, err := http.NewRequest("GET", ipURL, nil)
	if err != nil {
		fmt.Printf("Request creation error: %v\n", err)
		return
	}

	// 設定 Host header 為原始域名
	req.Host = "echo.free.beeceptor.com"
	req.Header.Set("Host", "echo.free.beeceptor.com")

	// 發送請求
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Request error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 讀取回應
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Response reading error: %v\n", err)
		return
	}

	fmt.Printf("Response status: %s\n", resp.Status)
	fmt.Printf("Response body: %s\n", string(body))
}
