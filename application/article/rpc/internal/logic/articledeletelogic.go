package logic

import (
	"context"

	"github.com/wangzhou-ccc/beyond/application/article/rpc/internal/svc"
	"github.com/wangzhou-ccc/beyond/application/article/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ArticleDeleteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewArticleDeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ArticleDeleteLogic {
	return &ArticleDeleteLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ArticleDeleteLogic) ArticleDelete(in *pb.ArticleDeleteRequest) (*pb.ArticleDeleteResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.ArticleDeleteResponse{}, nil
}
