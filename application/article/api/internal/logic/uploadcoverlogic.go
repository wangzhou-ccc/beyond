package logic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/wangzhou-ccc/beyond/application/article/api/internal/code"
	"github.com/wangzhou-ccc/beyond/application/article/api/internal/svc"
	"github.com/wangzhou-ccc/beyond/application/article/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	maxFileSize         = 10 << 20 // 10MB
	defaultBucketDomain = "https://beyond-article-zw.oss-cn-shanghai.aliyuncs.com"
)

type UploadCoverLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadCoverLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadCoverLogic {
	return &UploadCoverLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadCoverLogic) UploadCover(req *http.Request) (*types.UploadCoverResponse, error) {
	_ = req.ParseMultipartForm(maxFileSize)
	file, handler, err := req.FormFile("cover")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bucket, err := l.svcCtx.OssClient.Bucket(l.svcCtx.Config.Oss.BucketName)
	if err != nil {
		logx.Errorf("get bucket failed, err: %v", err)
		return nil, code.GetBucketErr
	}
	objectKey := genFilename(handler.Filename)
	err = bucket.PutObject(objectKey, file)
	if err != nil {
		logx.Errorf("put object failed, err: %v", err)
		return nil, code.PutBucketErr
	}

	return &types.UploadCoverResponse{CoverUrl: genFileURL(objectKey)}, nil
}

func genFilename(filename string) string {
	return fmt.Sprintf("%d_%s", time.Now().UnixMilli(), filename)
}

func genFileURL(objectKey string) string {
	return fmt.Sprintf("%s/%s", defaultBucketDomain, objectKey)
}
