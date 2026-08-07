package tts

import (
	"GopherAI/common/code"
	"GopherAI/common/tts"
	"GopherAI/controller"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	TTSRequest struct {
		Text string `json:"text,omitempty"`
	}
	TTSResponse struct {
		TaskID string `json:"task_id,omitempty"`
		controller.Response
	}
	QueryTTSResponse struct {
		TaskID     string `json:"task_id,omitempty"`
		TaskStatus string `json:"task_status,omitempty"`
		TaskResult string `json:"task_result,omitempty"`
		controller.Response
	}
)

type TTSServices struct {
	ttsService *tts.TTSService
}

func NewTTSServices() *TTSServices {
	return &TTSServices{
		ttsService: tts.NewTTSService(),
	}
}

// CreateTTSTask 创建TTS语音合成任务
// @Summary 创建TTS语音合成任务
// @Description 提交文本创建语音合成任务，返回任务ID供轮询查询
// @Tags 语音合成
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body tts.TTSRequest true "待合成文本"
// @Success 200 {object} tts.TTSResponse
// @Router /AI/chat/tts [post]
func CreateTTSTask(c *gin.Context) {
	tts := NewTTSServices()
	req := new(TTSRequest)
	res := new(TTSResponse)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	if req.Text == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	// 创建TTS任务并返回任务ID，由前端轮询查询结果
	taskID, err := tts.ttsService.CreateTTS(c, req.Text)
	if err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.TTSFail))
		return
	}

	res.Success()
	res.TaskID = taskID
	c.JSON(http.StatusOK, res)

}

// QueryTTSTask 查询TTS任务结果
// @Summary 查询TTS任务结果
// @Description 通过任务ID查询语音合成状态与结果URL
// @Tags 语音合成
// @Security ApiKeyAuth
// @Produce json
// @Param task_id query string true "任务ID"
// @Success 200 {object} tts.QueryTTSResponse
// @Router /AI/chat/tts/query [get]
func QueryTTSTask(c *gin.Context) {
	tts := NewTTSServices()
	res := new(QueryTTSResponse)
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	TTSQueryResponse, err := tts.ttsService.QueryTTSFull(c, taskID)
	if err != nil {
		log.Println("语音合成失败", err.Error())
		c.JSON(http.StatusOK, res.CodeOf(code.TTSFail))
		return
	}

	if len(TTSQueryResponse.TasksInfo) == 0 {
		c.JSON(http.StatusOK, res.CodeOf(code.TTSFail))
		return
	}

	res.Success()
	res.TaskID = TTSQueryResponse.TasksInfo[0].TaskID

	// 检查 TaskResult 是否为 nil，避免空指针异常
	if TTSQueryResponse.TasksInfo[0].TaskResult != nil {
		res.TaskResult = TTSQueryResponse.TasksInfo[0].TaskResult.SpeechURL
	}
	res.TaskStatus = TTSQueryResponse.TasksInfo[0].TaskStatus
	c.JSON(http.StatusOK, res)
}
