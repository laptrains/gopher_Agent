package image

import (
	"GopherAI/common/code"
	"GopherAI/controller"
	"GopherAI/service/image"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	RecognizeImageResponse struct {
		ClassName string `json:"class_name,omitempty"` // AI回答
		controller.Response
	}
)

// RecognizeImage 图像识别
// @Summary 图像识别
// @Description 上传一张图片，返回识别出的类别名称
// @Tags 图像
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "待识别的图片文件"
// @Success 200 {object} image.RecognizeImageResponse
// @Router /image/recognize [post]
func RecognizeImage(c *gin.Context) {
	res := new(RecognizeImageResponse)
	file, err := c.FormFile("image")
	if err != nil {
		log.Println("FormFile fail ", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	className, err := image.RecognizeImage(file)
	if err != nil {
		log.Println("RecognizeImage fail ", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.ClassName = className
	c.JSON(http.StatusOK, res)
}
