package logic

import (
	"context"

	"github.com/wangzhou-ccc/beyond/application/article/rpc/internal/svc"
	"github.com/wangzhou-ccc/beyond/application/article/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ArticleDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewArticleDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ArticleDetailLogic {
	return &ArticleDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ArticleDetailLogic) ArticleDetail(in *pb.ArticleDetailRequest) (*pb.ArticleDetailResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.ArticleDetailResponse{}, nil
}
