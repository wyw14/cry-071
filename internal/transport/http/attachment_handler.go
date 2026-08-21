package httpapi

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/domain"
	"github.com/wyw14/cry-071/internal/middleware"
)

func (h *handlers) uploadAttachment(c *gin.Context) {
	actor, _ := middleware.CurrentActor(c)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		respondError(c, domain.ValidationError{Field: "file", Message: "请选择要上传的文件"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		respondError(c, err)
		return
	}
	defer file.Close()
	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType == "" {
		buffer := make([]byte, 512)
		read, _ := file.Read(buffer)
		contentType = http.DetectContentType(buffer[:read])
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			respondError(c, err)
			return
		}
	}
	result, err := h.services.Attachment.Upload(c.Request.Context(), application.UploadAttachmentCommand{
		FeedbackID: c.Param("id"), FileName: fileHeader.Filename, ContentType: contentType, Size: fileHeader.Size,
		Reader: file, Actor: actor, RequestID: middleware.GetRequestID(c),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, result)
}

func (h *handlers) downloadAttachment(c *gin.Context) {
	actor, _ := middleware.CurrentActor(c)
	attachment, reader, err := h.services.Attachment.Open(c.Request.Context(), c.Param("id"), actor, c.Query("token"))
	if err != nil {
		respondError(c, err)
		return
	}
	defer reader.Close()
	c.Header("Content-Type", attachment.ContentType)
	c.Header("Content-Length", strconv.FormatInt(attachment.Size, 10))
	c.Header("Content-Disposition", `attachment; filename="download"`)
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, reader)
}
