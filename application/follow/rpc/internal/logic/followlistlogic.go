package logic

import (
	"context"
	"errors"
	"strconv"
	"time"

	A "github.com/IBM/fp-go/array"
	RIOE "github.com/IBM/fp-go/context/readerioeither"
	E "github.com/IBM/fp-go/either"
	F "github.com/IBM/fp-go/function"
	O "github.com/IBM/fp-go/option"
	T "github.com/IBM/fp-go/tuple"
	"github.com/wangzhou-ccc/beyond/application/follow/code"
	"github.com/wangzhou-ccc/beyond/application/follow/rpc/internal/model"
	"github.com/wangzhou-ccc/beyond/application/follow/rpc/internal/svc"
	"github.com/wangzhou-ccc/beyond/application/follow/rpc/internal/types"
	"github.com/wangzhou-ccc/beyond/application/follow/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type FollowListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFollowListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowListLogic {
	return &FollowListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// FollowList 关注列表 .
func (l *FollowListLogic) FollowList(in *pb.FollowListRequest) (*pb.FollowListResponse, error) {
	if in.UserId == 0 {
		return nil, code.UserIdEmpty
	}
	if in.PageSize == 0 {
		in.PageSize = types.DefaultPageSize
	}
	if in.Cursor == 0 {
		in.Cursor = time.Now().Unix()
	}

	var (
		err             error
		isCache, isEnd  bool
		lastId, cursor  int64
		followedUserIds []int64
		follows         []*model.Follow
		curPage         []*pb.FollowItem
	)

	followUserIds, _ := l.cacheFollowUserIds(l.ctx, in.UserId, in.Cursor, in.PageSize)
	if len(followUserIds) > 0 {
		isCache = true
		if followUserIds[len(followUserIds)-1] == -1 {
			followUserIds = followUserIds[:len(followUserIds)-1]
			isEnd = true
		}
		if len(followUserIds) == 0 {
			return &pb.FollowListResponse{}, nil
		}
		follows, err = l.svcCtx.FollowModel.FindByFollowedUserIds(l.ctx, followUserIds)
		if err != nil {
			l.Logger.Errorf("[FollowList] FollowModel.FindByFollowedUserIds error: %v req: %v", err, in)
			return nil, err
		}
		for _, follow := range follows {
			followedUserIds = append(followedUserIds, follow.FollowedUserID)
			curPage = append(curPage, &pb.FollowItem{
				Id:             follow.ID,
				FollowedUserId: follow.FollowedUserID,
				CreateTime:     follow.CreateTime.Unix(),
			})
		}
	} else {
		follows, err = l.svcCtx.FollowModel.FindByUserId(l.ctx, in.UserId, types.CacheMaxFollowCount)
		if err != nil {
			l.Logger.Errorf("[FollowList] FollowModel.FindByUserId error: %v req: %v", err, in)
			return nil, err
		}
		if len(follows) == 0 {
			return &pb.FollowListResponse{}, nil
		}
		var firstPageFollows []*model.Follow
		if len(follows) > int(in.PageSize) {
			firstPageFollows = follows[:in.PageSize]
		} else {
			firstPageFollows = follows
			isEnd = true
		}
		for _, follow := range firstPageFollows {
			followedUserIds = append(followedUserIds, follow.FollowedUserID)
			curPage = append(curPage, &pb.FollowItem{
				Id:             follow.ID,
				FollowedUserId: follow.FollowedUserID,
				CreateTime:     follow.CreateTime.Unix(),
			})
		}
	}
	if len(curPage) > 0 {
		pageLast := curPage[len(curPage)-1]
		lastId = pageLast.Id
		cursor = pageLast.CreateTime
		if cursor < 0 {
			cursor = 0
		}
		for k, follow := range curPage {
			if follow.CreateTime == in.Cursor && follow.Id == in.Id {
				curPage = curPage[k:]
				break
			}
		}
	}
	fc, err := l.svcCtx.FollowCountModel.FindByUserIds(l.ctx, followedUserIds)
	if err != nil {
		l.Logger.Errorf("[FollowList] FollowCountModel.FindByUserIds error: %v followedUserIds: %v", err, followedUserIds)
	}
	uidFansCount := make(map[int64]int)
	for _, f := range fc {
		uidFansCount[f.UserID] = f.FansCount
	}
	for _, cur := range curPage {
		cur.FansCount = int64(uidFansCount[cur.FollowedUserId])
	}
	ret := &pb.FollowListResponse{
		IsEnd:  isEnd,
		Cursor: cursor,
		Id:     lastId,
		Items:  curPage,
	}

	if !isCache {
		threading.GoSafe(func() {
			if len(follows) < types.CacheMaxFollowCount && len(follows) > 0 {
				follows = append(follows, &model.Follow{FollowedUserID: -1})
			}
			err = l.addCacheFollow(context.Background(), in.UserId, follows)
			if err != nil {
				logx.Errorf("addCacheFollow error: %v", err)
			}
		})
	}

	return ret, nil
}

func (l *FollowListLogic) FollowListFP(in *pb.FollowListRequest) (*pb.FollowListResponse, error) {
	cacheFollowUserIds := T.Tupled3(RIOE.Eitherize3(l.cacheFollowUserIds))
	findByFollowedUserIds := RIOE.Eitherize1(l.svcCtx.FollowModel.FindByFollowedUserIds)

	inputPipe := F.Pipe2(
		T.MakeTuple3(in.UserId, in.Cursor, in.PageSize),
		func(t T.Tuple3[int64, int64, int64]) E.Either[error, T.Tuple3[int64, int64, int64]] {
			if t.F1 == 0 {
				return E.Left[T.Tuple3[int64, int64, int64], error](code.UserIdEmpty)
			}

			return E.Right[error](t)
		},
		E.Chain(func(t T.Tuple3[int64, int64, int64]) E.Either[error, T.Tuple3[int64, int64, int64]] {
			var (
				pageSize = t.F2
				cursor   = t.F3
			)

			if pageSize == 0 {
				pageSize = types.DefaultPageSize
			}

			if cursor == 0 {
				cursor = time.Now().Unix()
			}

			return E.Right[error](T.MakeTuple3(t.F1, cursor, pageSize))
		}),
	)

	followsPipe := F.Pipe1(
		RIOE.FromEither(inputPipe),
		RIOE.Chain(func(in T.Tuple3[int64 /* userId */, int64 /* cursor */, int64 /* pageSize */]) RIOE.ReaderIOEither[T.Tuple2[[]*model.Follow, bool /* is end */]] {
			return F.Pipe4(
				in,
				cacheFollowUserIds,
				RIOE.Chain(func(userIds []int64) RIOE.ReaderIOEither[[]int64] {
					if len(userIds) == 0 {
						return RIOE.Left[[]int64](errors.New("follow user id missing in cache"))
					}

					return RIOE.Right(userIds[:len(userIds)-1])
				}),
				RIOE.Chain(func(userIds []int64) RIOE.ReaderIOEither[T.Tuple2[[]*model.Follow, bool]] {
					if len(userIds) == 0 {
						return RIOE.Right(T.MakeTuple2([]*model.Follow{}, true))
					} else {
						return RIOE.SequenceParT2(findByFollowedUserIds(userIds), RIOE.Right[bool](false))
					}
				}),
				RIOE.OrElse(func(err error) RIOE.ReaderIOEither[T.Tuple2[[]*model.Follow, bool]] {
					if errors.Is(err, code.UserIdEmpty) {
						return RIOE.Left[T.Tuple2[[]*model.Follow, bool]](err)
					}

					return F.Pipe2(
						T.MakeTuple2(in.F1, types.CacheMaxFollowCount),
						T.Tupled2(RIOE.Eitherize2(l.svcCtx.FollowModel.FindByUserId)),
						RIOE.Chain(func(follows []*model.Follow) RIOE.ReaderIOEither[T.Tuple2[[]*model.Follow, bool]] {
							if len(follows) > int(in.F3) {
								return RIOE.Right(T.MakeTuple2(follows[:in.F3], false))
							} else {
								return RIOE.Right(T.MakeTuple2(follows, true))
							}
						}),
					)
				}),
			)
		}),
	)

	composeOut := F.Pipe2(
		followsPipe,
		RIOE.Chain(func(i T.Tuple2[[]*model.Follow, bool /* is end */]) RIOE.ReaderIOEither[T.Tuple2[[]*pb.FollowItem, bool /* is end */]] {
			followCounts := F.Pipe1(
				F.Pipe1(i.F1, A.Map(func(f *model.Follow) int64 { return f.FollowedUserID })),
				RIOE.Eitherize1(l.svcCtx.FollowCountModel.FindByUserIds),
			)

			return F.Pipe1(
				RIOE.SequenceT2(RIOE.Of(i.F1), followCounts),
				RIOE.Chain(func(t T.Tuple2[[]*model.Follow, []*model.FollowCount]) RIOE.ReaderIOEither[T.Tuple2[[]*pb.FollowItem, bool]] {
					return F.Pipe4(
						t.F1,
						A.Map(func(f *model.Follow) T.Tuple2[*model.Follow, O.Option[int64]] {
							cnt := F.Pipe2(
								t.F2,
								A.FindFirst(func(fc *model.FollowCount) bool { return fc.UserID == f.FollowedUserID }),
								O.Map(func(fc *model.FollowCount) int64 { return int64(fc.FansCount) }),
							)

							return T.MakeTuple2(f, cnt)
						}),
						A.Map(func(t T.Tuple2[*model.Follow, O.Option[int64]]) *pb.FollowItem {
							return &pb.FollowItem{
								Id:             t.F1.ID,
								FollowedUserId: t.F1.FollowedUserID,
								FansCount:      O.GetOrElse(F.Constant[int64](0))(t.F2),
								CreateTime:     t.F1.CreateTime.Unix(),
							}
						}),
						func(items []*pb.FollowItem) T.Tuple2[[]*pb.FollowItem, bool] {
							return T.MakeTuple2(items, i.F2)
						},
						RIOE.Of[T.Tuple2[[]*pb.FollowItem, bool]],
					)
				}),
			)
		}),
		RIOE.Chain(func(in T.Tuple2[[]*pb.FollowItem, bool]) RIOE.ReaderIOEither[*pb.FollowListResponse] {
			lastFollowItem := F.Pipe2(
				in.F1,
				A.Last[*pb.FollowItem],
				O.Fold(
					func() T.Tuple2[int64, int64] { return T.MakeTuple2[int64, int64](0, 0) },
					func(f *pb.FollowItem) T.Tuple2[int64, int64] { return T.MakeTuple2(f.Id, f.CreateTime) },
				),
			)

			return RIOE.Of(&pb.FollowListResponse{
				Items:  in.F1,
				Cursor: lastFollowItem.F2,
				IsEnd:  in.F2,
				Id:     lastFollowItem.F1,
			})
		}),
	)

	return E.Unwrap(composeOut(l.ctx)())
}

func (l *FollowListLogic) cacheFollowUserIds(ctx context.Context, userId, cursor, pageSize int64) ([]int64, error) {
	key := userFollowKey(userId)
	b, err := l.svcCtx.BizRedis.ExistsCtx(ctx, key)
	if err != nil {
		logx.Errorf("[cacheFollowUserIds] BizRedis.ExistsCtx error: %v", err)
	}
	if b {
		err = l.svcCtx.BizRedis.ExpireCtx(ctx, key, types.UserFollowExpireTime)
		if err != nil {
			logx.Errorf("[cacheFollowUserIds] BizRedis.ExpireCtx error: %v", err)
		}
	}
	pairs, err := l.svcCtx.BizRedis.ZrevrangebyscoreWithScoresAndLimitCtx(ctx, key, 0, cursor, 0, int(pageSize))
	if err != nil {
		logx.Errorf("[cacheFollowUserIds] BizRedis.ZrevrangebyscoreWithScoresAndLimitCtx error: %v", err)
		return nil, err
	}
	var uids []int64
	for _, pair := range pairs {
		uid, err := strconv.ParseInt(pair.Key, 10, 64)
		if err != nil {
			logx.Errorf("[cacheFollowUserIds] strconv.ParseInt error: %v", err)
			continue
		}
		uids = append(uids, uid)
	}

	return uids, nil
}

func (l *FollowListLogic) addCacheFollow(ctx context.Context, userId int64, follows []*model.Follow) error {
	if len(follows) == 0 {
		return nil
	}
	key := userFollowKey(userId)
	for _, follow := range follows {
		var score int64
		if follow.FollowedUserID == -1 {
			score = 0
		} else {
			score = follow.CreateTime.Unix()
		}
		_, err := l.svcCtx.BizRedis.ZaddCtx(ctx, key, score, strconv.FormatInt(follow.FollowedUserID, 10))
		if err != nil {
			logx.Errorf("[addCacheFollow] BizRedis.ZaddCtx error: %v", err)
			return err
		}
	}

	return l.svcCtx.BizRedis.ExpireCtx(ctx, key, types.UserFollowExpireTime)
}
