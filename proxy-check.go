package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	URL "net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/mmpx12/optionparser"
	"github.com/schollz/progressbar/v3"
)

var (
	Proxies    []string
	mu         = &sync.Mutex{}
	valid      []string
	checkmax   bool
	counter    int32
	maxvalid   int
	disableBar bool
	noProto    bool
	delete     bool
	file       string
	version    = "1.1.2"
)

func ProxyTest(client *http.Client, proxy, urlTarget, timeout string) bool {
	timeouts, _ := strconv.Atoi(timeout)
	ctx, cncl := context.WithTimeout(context.Background(), time.Second*time.Duration(timeouts))
	defer cncl()
	proxyURL, _ := URL.Parse(proxy)
	client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlTarget, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")
	resp, err := client.Do(req)

	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if disableBar {
			fmt.Println("\033[31m[X] ", proxy, "\033[0m")
		}
		return false
	}

	if resp.StatusCode != http.StatusOK {
		if disableBar {
			fmt.Println("\033[31m[X] ", proxy, "\033[0m")
		}
		return false
	}
	if disableBar {
		fmt.Println("\033[32m[√] ", proxy, "\033[0m")
	}
	mu.Lock()
	valid = append(valid, proxy)
	mu.Unlock()
	return true
}

func readLines(http, socks4, socks5, all bool) {
	file, err := os.Open(file)
	if err != nil {
		print(err.Error(), "\n")
		os.Exit(1)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var proto = []string{"http://", "socks4://", "socks5://"}
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "http://") {
			if http == true || all == true {
				Proxies = append(Proxies, scanner.Text())
			}
		} else if strings.HasPrefix(scanner.Text(), "socks4://") {
			if socks4 == true || all == true {
				Proxies = append(Proxies, scanner.Text())
			}
		} else if strings.HasPrefix(scanner.Text(), "socks5://") {
			if socks5 == true || all == true {
				Proxies = append(Proxies, scanner.Text())
			}
		} else if noProto {
			var ipv4 = regexp.MustCompile(`^(((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.|$|:[0-9]{1,5})){4})`)
			if ipv4.MatchString(scanner.Text()) {
				proxy := scanner.Text()
				for _, i := range proto {
					Proxies = append(Proxies, i+proxy)
				}
			}
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
	var url, goroutine, timeout, urlfile, output, nbrvalid string
	op := optionparser.NewOptionParser()
	op.Banner = "Proxy tester\n\nUsage:\n"
	op.On("-s", "--socks4", "Test socks4 proxies", &socks4)
	op.On("-S", "--socks5", "Test socks5 proxies", &socks5)
	op.On("-H", "--http", "Test http proxies", &httpp)
	op.On("-r", "--randomize-file", "Shuffle proxies files", &random)
	op.On("-t", "--thread NBR", "Number of threads", &goroutine)
	op.On("-T", "--timeout SEC", "Set timeout (seconds)", &timeout)
	op.On("-u", "--url TARGET", "set URL for testing proxies", &url)
	op.On("-f", "--proxies-file FILE", "files with proxies (proto://ip:port)", &file)
	op.On("-m", "--max-valid NBR", "Stop when NBR valid proxies are found", &nbrvalid)
	op.On("-i", "--ip", "Test all proto if no proto is specified in input", &noProto)
	op.On("-U", "--proxies-url URL", "url with proxies file", &urlfile)
	op.On("-p", "--dis-progressbar", "Disable progress bar", &disableBar)
	op.On("-g", "--github", "use github.com/mmpx12/proxy-list", &github)
	op.On("-o", "--output FILE", "File to write valid proxies", &output)
	op.On("-v", "--version", "Print version and exit", &printversion)
	op.Exemple("proxy-check -r -m 30 --socks5 -o valid-socks5.txt  -g")
	op.Exemple("proxy-check -m 30 -o valid.txt -U 'https://raw.githubusercontent.com/mmpx12/proxy-list/master/proxies.txt'")
	op.Exemple("proxy-check -u ipinfo.io -T 6 /path/to/proxy")
	err := op.Parse()
	if err != nil {
		print(err.Error(), "\n")
		os.Exit(1)
	}
	op.Logo("Proxy-check", "smslant", nologo)

	if printversion {
		fmt.Println("version:", version)
		os.Exit(1)
	}

	if err != nil || len(os.Args) == 1 {
		op.Help()
		os.Exit(1)
	}

	if nbrvalid != "" {
		maxvalid, _ = strconv.Atoi(nbrvalid)
		checkmax = true
	} else {
		maxvalid = 0
	}

	opts := strings.Join(strings.Fields(strings.TrimSpace(strings.Join(op.Extra, ""))), " ")
	if (opts != "" || !github || urlfile == "") && file == "" {
		file = opts
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
		r, err := http.Get(urlfile)
		if err != nil {
			print(err.Error(), "\n")
			os.Exit(1)
		}
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
		timeout = "3"
	}

	if goroutine == "" {
		goroutine = "50"
	}

	var wg = sync.WaitGroup{}
	goroutines, _ := strconv.Atoi(goroutine)
	maxGoroutines := goroutines
	guard := make(chan struct{}, maxGoroutines)
	readLines(httpp, socks4, socks5, all)

	if random {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(Proxies),
			func(i, j int) {
				Proxies[i], Proxies[j] = Proxies[j], Proxies[i]
			})
	}

	var size int
	if maxvalid > 0 {
		size = maxvalid
	} else {
		size = len(Proxies)
	}
	bar := progressbar.NewOptions(size,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetVisibility(!disableBar),
		progressbar.OptionSetDescription("[green]"+strconv.Itoa(int(counter))),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]━[reset]",
			SaucerHead:    "[green][dark_gray]",
			SaucerPadding: "━",
			BarStart:      "[light_gray][[dark_gray]",
			BarEnd:        "[light_gray]][reset]",
		}))

	var client *http.Client
	for i, j := range Proxies {

		bar.Describe("[green]" + strconv.Itoa(int(counter)) + "[light_gray]|[red]" + strconv.Itoa(i-int(counter)) + "[light_gray]|[yellow]" + strconv.Itoa(size) + "[reset]")
		mu.Lock()
		if checkmax && counter >= int32(maxvalid) {
			writeResult(output, file)
			bar.Finish()
			fmt.Println()
			fmt.Println("\033[4mValid proxies:\033[0m\n")
			for _, v := range valid {
				fmt.Println(v)
			}
			if delete {
				os.Remove(file)
			}
			os.Exit(0)
		}
		mu.Unlock()
		guard <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			var res bool
			res = ProxyTest(client, j, url, timeout)
			if res == true {
				mu.Lock()
				bar.Add(1)
				atomic.AddInt32(&counter, 1)
				mu.Unlock()
			}
			<-guard
		}()
	}
	wg.Wait()
	bar.Finish()
	writeResult(output, file)
	fmt.Println()
	fmt.Println("\033[4mValid proxies:\033[0m\n")
	for _, v := range valid {
		fmt.Println(v)
	}
	if delete {
		os.Remove(file)
	}
}
