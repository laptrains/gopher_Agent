package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

//wttr.in JSON 响应结构

type WttrResponse struct {
	CurrentCondition []struct {
		TempC         string `json:"temp_C"`
		Humidity      string `json:"humidity"`
		WindspeedKmph string `json:"windspeedKmph"`
		WeatherDesc   []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
	} `json:"current_condition"`

	NearestArea []struct {
		AreaName []struct {
			Value string `json:"value"`
		} `json:"areaName"`
	} `json:"nearest_area"`
}

//统一对外天气结构

type WeatherResponse struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"windSpeed"`
}

//Weather API Client

type WeatherAPIClient struct{}

func NewWeatherAPIClient() *WeatherAPIClient {
	return &WeatherAPIClient{}
}

func (c *WeatherAPIClient) GetWeather(ctx context.Context, city string) (*WeatherResponse, error) {
	apiURL := fmt.Sprintf(
		"https://wttr.in/%s?format=j1&lang=zh",
		city,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	var wttrResp WttrResponse
	if err := json.Unmarshal(body, &wttrResp); err != nil {
		return nil, fmt.Errorf("json parse failed: %w", err)
	}

	if len(wttrResp.CurrentCondition) == 0 {
		return nil, fmt.Errorf("no weather data")
	}

	cc := wttrResp.CurrentCondition[0]
	//解析温度
	temp, _ := strconv.ParseFloat(cc.TempC, 64)
	//解析湿度
	humidity, _ := strconv.Atoi(cc.Humidity)
	//解析风速
	wind, _ := strconv.ParseFloat(cc.WindspeedKmph, 64)

	location := city
	if len(wttrResp.NearestArea) > 0 &&
		len(wttrResp.NearestArea[0].AreaName) > 0 {
		location = wttrResp.NearestArea[0].AreaName[0].Value
	}

	condition := "未知"
	if len(cc.WeatherDesc) > 0 {
		condition = cc.WeatherDesc[0].Value
	}

	return &WeatherResponse{
		Location:    location,
		Temperature: temp,
		Condition:   condition,
		Humidity:    humidity,
		WindSpeed:   wind,
	}, nil
}

//统一对外股票行情结构

type StockResponse struct {
	Name      string  `json:"name"`
	Code      string  `json:"code"`
	Price     float64 `json:"price"`
	PrevClose float64 `json:"prevClose"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Change    float64 `json:"change"`
	ChangePct float64 `json:"changePct"`
}

//Stock API Client（腾讯行情接口 qt.gtimg.cn，免费无需 key，返回 GBK 编码文本）

type StockAPIClient struct{}

func NewStockAPIClient() *StockAPIClient {
	return &StockAPIClient{}
}

// normalizeStockCode 自动补全市场前缀：6 开头沪市、0/3 开头深市、4/8 开头北交所
// 已带前缀（如 sh600519、hk00700、usAAPL）则原样使用
func normalizeStockCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if len(code) == 6 && code[0] >= '0' && code[0] <= '9' {
		switch code[0] {
		case '6':
			return "sh" + code
		case '0', '3':
			return "sz" + code
		case '4', '8':
			return "bj" + code
		}
	}
	return code
}

func (c *StockAPIClient) GetStockPrice(ctx context.Context, code string) (*StockResponse, error) {
	fullCode := normalizeStockCode(code)
	apiURL := fmt.Sprintf("https://qt.gtimg.cn/q=%s", fullCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	// 接口返回 GBK 编码文本：v_sh600519="1~贵州茅台~600519~...";
	utf8Body, err := io.ReadAll(transform.NewReader(bytes.NewReader(body), simplifiedchinese.GBK.NewDecoder()))
	if err != nil {
		return nil, fmt.Errorf("gbk decode failed: %w", err)
	}

	text := strings.TrimSpace(string(utf8Body))
	eq := strings.Index(text, "=")
	if eq < 0 {
		return nil, fmt.Errorf("unexpected response format")
	}
	payload := strings.Trim(text[eq+1:], "\"; \r\n\t")
	parts := strings.Split(payload, "~")
	// 字段位置（~ 分隔）：1=名称 2=代码 3=现价 4=昨收 5=今开 31=涨跌 32=涨跌% 33=最高 34=最低
	if len(parts) < 35 || parts[1] == "" {
		return nil, fmt.Errorf("no stock data for %s", code)
	}

	parse := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	return &StockResponse{
		Name:      parts[1],
		Code:      parts[2],
		Price:     parse(parts[3]),
		PrevClose: parse(parts[4]),
		Open:      parse(parts[5]),
		Change:    parse(parts[31]),
		ChangePct: parse(parts[32]),
		High:      parse(parts[33]),
		Low:       parse(parts[34]),
	}, nil
}

/*
	========================
	MCP Server
	========================
*/

func NewMCPServer() *server.MCPServer {
	weatherClient := NewWeatherAPIClient()
	stockClient := NewStockAPIClient()

	mcpServer := server.NewMCPServer(
		"weather-query-server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	mcpServer.AddTool(
		mcp.NewTool(
			"get_weather",
			mcp.WithDescription("获取指定城市的天气信息"),
			mcp.WithString(
				"city",
				mcp.Description("城市名称，如 Beijing、上海"),
				mcp.Required(),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			city, ok := args["city"].(string)
			if !ok || city == "" {
				return nil, fmt.Errorf("invalid city argument")
			}

			weather, err := weatherClient.GetWeather(ctx, city)
			if err != nil {
				return nil, err
			}

			resultText := fmt.Sprintf(
				"城市: %s\n温度: %.1f°C\n天气: %s\n湿度: %d%%\n风速: %.1f km/h",
				weather.Location,
				weather.Temperature,
				weather.Condition,
				weather.Humidity,
				weather.WindSpeed,
			)

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: resultText,
					},
				},
			}, nil
		},
	)

	mcpServer.AddTool(
		mcp.NewTool(
			"get_stock_price",
			mcp.WithDescription("查询股票实时行情（A股直接填6位代码，如 600519；也可带市场前缀，如 sh600519、hk00700、usAAPL）"),
			mcp.WithString(
				"code",
				mcp.Description("股票代码，如 600519、000001、sh600519"),
				mcp.Required(),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			code, ok := args["code"].(string)
			if !ok || code == "" {
				return nil, fmt.Errorf("invalid code argument")
			}

			stock, err := stockClient.GetStockPrice(ctx, code)
			if err != nil {
				return nil, err
			}

			resultText := fmt.Sprintf(
				"股票: %s (%s)\n现价: %.2f\n涨跌: %+.2f (%+.2f%%)\n今开: %.2f\n昨收: %.2f\n最高: %.2f\n最低: %.2f",
				stock.Name,
				stock.Code,
				stock.Price,
				stock.Change,
				stock.ChangePct,
				stock.Open,
				stock.PrevClose,
				stock.High,
				stock.Low,
			)

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: resultText,
					},
				},
			}, nil
		},
	)

	return mcpServer
}

// StartServer 启动MCP服务器
// httpAddr: HTTP服务器监听的地址（例如":8080"）
func StartServer(httpAddr string) error {
	mcpServer := NewMCPServer()
	//把mcpServer注册到httpServer
	httpServer := server.NewStreamableHTTPServer(mcpServer)
	log.Printf("HTTP MCP server listening on %s/mcp", httpAddr)
	return httpServer.Start(httpAddr)
}
