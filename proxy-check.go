package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mmpx12/optionparser"
	"h12.io/socks"
)

var (
	Proxies  []string
	mu       = &sync.Mutex{}
	valid    []string
	checkmax bool
	counter  int32
	maxvalid int
	delete   bool
	version  = "1.0.0"
)

func HttpTest(proxy, urlTarget, timeout string) bool {
	proxyURL, _ := url.Parse(proxy)
	timeouts, _ := strconv.Atoi(timeout)
	ctx, cncl := context.WithTimeout(context.Background(), time.Second*time.Duration(timeouts))
	defer cncl()
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: transport}
	request, _ := http.NewRequestWithContext(ctx, "GET", urlTarget, nil)
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("\033[31m[X] ", proxy, "\033[0m")
		return false
	}

	if response.StatusCode != http.StatusOK {
		fmt.Println("\033[31m[X] ", proxy, "\033[0m")
		return false
	}

	fmt.Println("\033[32m[√] ", proxy, "\033[0m")
	mu.Lock()
	valid = append(valid, proxy)
	mu.Unlock()
	return true
}

func SocksTest(proxy, urlTarget, timeout string) bool {
	dialSocksProxy := socks.Dial(proxy + "?timeout=" + timeout + "s")
	tr := &http.Transport{Dial: dialSocksProxy}
	httpClient := &http.Client{Transport: tr}
	resp, err := httpClient.Get(urlTarget)
	if err != nil {
		fmt.Println("\033[31m[X] ", proxy, "\033[0m")
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("\033[31m[X] ", proxy, "\033[0m")
		return false
	}
	fmt.Println("\033[32m[√] ", proxy, "\033[0m")
	mu.Lock()
	valid = append(valid, proxy)
	mu.Unlock()
	return true
}

func readLines(path string, http, socks4, socks5, all bool) {
	file, err := os.Open(path)
	if err != nil {
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "http://") && (http == true || all == true) {
			Proxies = append(Proxies, scanner.Text())
		} else if strings.HasPrefix(scanner.Text(), "socks4://") && (socks4 == true || all == true) {
			Proxies = append(Proxies, scanner.Text())
		} else if strings.HasPrefix(scanner.Text(), "socks5://") && (socks5 == true || all == true) {
			Proxies = append(Proxies, scanner.Text())
		}
	}
}

func writeResult(output, file string) {
	if output != "" {
		ff, _ := os.OpenFile(output, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		defer ff.Close()
		for _, i := range valid {
			fmt.Fprintln(ff, i)
		}
	}

	if delete {
		os.Remove(file)
	}
}

func main() {
	var nologo, socks5, socks4, httpp, all, random, github, printversion bool
	var file, url, goroutine, timeout, urlfile, output, nbrvalid string
	//var timeout int
	op := optionparser.NewOptionParser()
	op.Banner = "Proxy tester\n\nUsage:\n"
	op.On("-s", "--socks4", "Test socks4 proxies", &socks4)
	op.On("-S", "--socks5", "Test socks5 proxies", &socks5)
	op.On("-H", "--http", "Test http proxies", &httpp)
	op.On("-r", "--randomize-file", "Shuffle proxies files", &random)
	op.On("-t", "--thread NBR", "Number of threads", &goroutine)
	op.On("-T", "--timeout SEC", "Set timeout (seconds)", &timeout)
	op.On("-u", "--url TARGET", "set URL for testing proxies", &url)
	op.On("-f", "--proxies-file FILE", "files with proxies (proto://ip:port)", file)
	op.On("-m", "--max-valid NBR", "Stop when NBR valid proxies are found", &nbrvalid)
	op.On("-U", "--proxies-url URL", "url with proxies file", &urlfile)
	op.On("-g", "--github", "use github.com/mmpx12/proxy-list", &github)
	op.On("-o", "--output FILE", "File to write valid proxies", &output)
	op.On("-v", "--version", "Print version and exit", &printversion)
	op.Exemple("proxy-check -r -m 30 --socks5 -o valid-socks5.txt  -g")
	op.Exemple("proxy-check -m 30 -o valid.txt -U 'https://raw.githubusercontent.com/mmpx12/proxy-list/master/proxies.txt'")
	op.Exemple("proxy-check -u ipinfo.io -T 6 /path/to/proxy")
	err := op.Parse()
	op.Logo("Proxy-check", "smslant", nologo)

	if printversion {
		fmt.Println("version:", version)
		os.Exit(1)
	}

	if err != nil || len(os.Args) == 1 {
		op.Help()
	}

	if nbrvalid != "" {
		maxvalid, _ = strconv.Atoi(nbrvalid)
		checkmax = true
	} else {
		maxvalid = 0
	}

	if strings.Join(op.Extra, "") != "" || !github || urlfile == "" {
		file = strings.Join(op.Extra, "")
	} else if file == "" {
		fmt.Println("Error: Need file or url with proxies")
		os.Exit(1)
	}

	if github {
		delete = true
		urlfile = "https://raw.githubusercontent.com/mmpx12/proxy-list/master/proxies.txt"
	}

	if urlfile != "" {
		if !strings.Contains(urlfile, "http://") && !strings.Contains(urlfile, "https://") {
			urlfile = "http://" + urlfile
		}
		delete = true
		b := make([]byte, 5)
		rand.Seed(time.Now().UnixNano())
		rand.Read(b)
		file = fmt.Sprintf("proxies-%x.txt", b)
		r, _ := http.Get(urlfile)
		defer r.Body.Close()
		f, _ := os.Create(file)
		defer f.Close()
		r.Write(f)
	}

	if !httpp && !socks4 && !socks5 {
		all = true
	}

	if url == "" {
		url = "http://checkip.amazonaws.com"
	} else if !strings.Contains(url, "http://") && !strings.Contains(url, "https://") {
		url = "http://" + url
	}

	if timeout == "" {
		timeout = "5"
	}

	if goroutine == "" {
		goroutine = "50"
	}

	var wg = sync.WaitGroup{}
	goroutines, _ := strconv.Atoi(goroutine)
	maxGoroutines := goroutines
	guard := make(chan struct{}, maxGoroutines)
	readLines(file, httpp, socks4, socks5, all)

	if random {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(Proxies),
			func(i, j int) {
				Proxies[i], Proxies[j] = Proxies[j], Proxies[i]
			})
	}

	for _, j := range Proxies {
		mu.Lock()
		if checkmax && counter >= int32(maxvalid) {
			writeResult(output, file)
			os.Exit(0)
		}
		mu.Unlock()
		guard <- struct{}{}
		wg.Add(1)
		go func(j string, url string, timeOut string) {
			var res bool
			if strings.HasPrefix(j, "http://") {
				res = HttpTest(j, url, timeOut)
			} else {
				res = SocksTest(j, url, timeOut)
			}
			if res == true {
				mu.Lock()
				atomic.AddInt32(&counter, 1)
				mu.Unlock()
			}
			<-guard
			wg.Done()
		}(j, url, timeout)
	}
	wg.Wait()
}
