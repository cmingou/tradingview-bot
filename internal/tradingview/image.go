package tradingview

import (
	"fmt"
	"github.com/pkg/errors"
	tb "gopkg.in/tucnak/telebot.v2"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func tradingviewWidget(symbols, desc, timeRange string) string {
	return fmt.Sprintf(`
<!-- TradingView Widget BEGIN -->
<div class="tradingview-widget-container">
  <div id="tradingview_9efce"></div>
  <div class="tradingview-widget-copyright">图表仅供讨论使用，资料由TradingView提供</div>
  <script type="text/javascript" src="https://s3.tradingview.com/tv.js"></script>
  <script type="text/javascript">
  new TradingView.MediumWidget(
  {
  "symbols": [
    [
      "%s",
      "%s%s"
    ]
  ],
  "chartOnly": false,
  "width": 1000,
  "height": 400,
  "locale": "zh_CN",
  "colorTheme": "dark",
  "gridLineColor": "#2A2E39",
  "trendLineColor": "#1976D2",
  "fontColor": "#787B86",
  "underLineColor": "rgba(55, 166, 239, 0.15)",
  "isTransparent": false,
  "autosize": false,
  "container_id": "tradingview_9efce"
}
  );
  </script>
</div>
<!-- TradingView Widget END -->
`, desc, symbols, timeRange)
}

func tradingviewDetailWidget(symbols, timeRange string) string {
	return fmt.Sprintf(`
<!-- TradingView Widget BEGIN -->
<div class="tradingview-widget-container">
  <div id="tradingview_f9dfa"></div>
  <div class="tradingview-widget-copyright">图表仅供讨论使用，资料由TradingView提供</div>
  <script type="text/javascript" src="https://s3.tradingview.com/tv.js"></script>
  <script type="text/javascript">
  new TradingView.widget(
  {
  "width": 1000,
  "height": 400,
  "symbol": "%v",
  "timezone": "America/New_York",
  "theme": "dark",
  "style": "1",
  "locale": "zh_CN",
  "toolbar_bg": "#f1f3f6",
  "enable_publishing": false,
  "hide_top_toolbar": true,
  "range": "%v",
  "allow_symbol_change": true,
  "save_image": false,
  "studies": [
    "MASimple@tv-basicstudies"
  ],
  "container_id": "tradingview_f9dfa"
}
  );
  </script>
</div>
<!-- TradingView Widget END -->
`, symbols, timeRange)
}

type StockImageOptions struct {
	Symbol      string
	Description string
	Time        string

	// BinaryPath the path to your capture-website binary. REQUIRED
	//
	// Must be absolute path e.g /usr/local/bin/capture-website
	BinaryPath string

	Input             string
	Output            string
	Dir               string
	Html              string
	Format            string
	Width             int
	Height            int
	Delay             int
	Overwrite         bool
	Darkmode          bool
	TechnicalAnalysis bool
}

func (s *StockImageOptions) FilePath() string {
	return s.Dir + "/" + s.Output + "." + s.Format
}

func (s *StockImageOptions) FileName() string {
	return s.Output + "." + s.Format
}

func (s *StockImageOptions) CountdownToDel() {
	go func() {
		// countdown
		timer := time.NewTimer(time.Second * 20)
		<-timer.C
		// delete images
		if err := os.Remove(s.FilePath()); err != nil {
			fmt.Printf("%v\n", err)
		}
	}()
}

func (s *StockImageOptions) GenerateImage() error {
	arr, err := buildParams(s, s.TechnicalAnalysis)
	if err != nil {
		return err
	}

	if s.BinaryPath == "" {
		sysType := runtime.GOOS
		switch sysType {
		case "linux":
			s.BinaryPath = "/usr/bin/capture-website"
		case "darwin":
			s.BinaryPath = "/usr/local/bin/capture-website"
		default:
			return errors.Errorf("Not support this OS: %v", sysType)
		}
	}

	cmd := exec.Command(s.BinaryPath, arr...)

	if s.Html != "" {
		cmd.Stdin = strings.NewReader(s.Html)
	}

	_, err = cmd.CombinedOutput()

	return err
}

func buildParams(options *StockImageOptions, technicalAnalysis bool) ([]string, error) {
	a := []string{}

	options.Format = "png"

	if options.Symbol == "" {
		return []string{}, errors.New("Must provide symbol")
	}

	if options.Description == "" {
		return []string{}, errors.New("Must provide description")
	}

	if technicalAnalysis {
		options.Html = tradingviewDetailWidget(options.Symbol, options.Time[1:])
	} else {
		options.Html = tradingviewWidget(options.Description, options.Symbol, options.Time)
	}

	if options.Input == "" {
		return []string{}, errors.New("Must provide input")
	}

	if options.Height != 0 {
		a = append(a, "--height")
		a = append(a, strconv.Itoa(options.Height))
	}

	if options.Width != 0 {
		a = append(a, "--width")
		a = append(a, strconv.Itoa(options.Width))
	}

	if options.Delay != 0 {
		a = append(a, "--delay")
		a = append(a, strconv.Itoa(options.Delay))
	}

	if options.Overwrite {
		a = append(a, "--overwrite")
	}

	if options.Darkmode {
		a = append(a, "--dark-mode")
	}

	// 如果设置了 URL，则优先使用 URL
	if options.Input != "-" {
		// 如果使用 URL 则需要将 Html 参数置空
		options.Html = ""
	}

	//a = append(a, options.Input)

	if options.Output == "" {
		return nil, errors.Errorf("Must provide output")
		//a = append(a, "-")
	} else {
		a = append(a, "--output")
		a = append(a, options.FilePath())
	}

	fmt.Printf("parameters:\n%v\n", a)
	return a, nil
}

func SearchAndSendStockImage(b *tb.Bot, m *tb.Message, symbol, timeRange string, delFile, technicalAnalysis bool) {
	var err error
	if symbol == "" {
		// Did not add stock id
		mReply, _ := b.Reply(m, "请输入股票代号, 例如: /chart AAPL")

		// Waiting to delete
		go func() {
			time.Sleep(time.Second * 6)
			err = b.Delete(mReply)
			if err != nil {
				fmt.Printf("%v\n", err)
			}

			err = b.Delete(m)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		}()
		return
	}

	imgName := strconv.Itoa(int(m.Unixtime)) + "-" + strconv.Itoa(m.ID) + "-" + symbol + "-" + m.Sender.Username

	//Generate stock image
	s := StockImageOptions{
		Symbol:            symbol,
		Description:       symbol,
		Time:              timeRange,
		Input:             "-",
		Output:            imgName,
		Dir:               "./img",
		Width:             1015,
		Height:            400,
		Delay:             4,
		Overwrite:         true,
		Darkmode:          true,
		TechnicalAnalysis: technicalAnalysis,
	}

	if err := s.GenerateImage(); err != nil {
		fmt.Printf("%v\n", err)
	}
	p := &tb.Photo{
		File: tb.File{
			FileLocal: "./img/" + imgName + ".png",
		},
		Width:  1015,
		Height: 400,
	}

	if _, err := b.Reply(m, p); err != nil {
		fmt.Printf("%v\n", err)
	}

	s.CountdownToDel()

	time.Sleep(time.Second)

	// Delete request user command
	if delFile {
		if err = b.Delete(m); err != nil {
			fmt.Printf("%v\n", err)
		}
	}
}
