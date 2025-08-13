package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"bdo_calc_go/internal/config"
	"bdo_calc_go/internal/handler"
	"bdo_calc_go/internal/repo"
	"bdo_calc_go/internal/router"
	"bdo_calc_go/internal/service"
	"bdo_calc_go/pkg/logger"
)

func main() {
	// 설정/로거 초기화
	cfg := config.Load()
	logg := logger.New()

	// 의존성 생성
	userRepo := repo.NewUserRepoInMemory()
	userSvc := service.NewUserService(userRepo, logg)
	userH := handler.NewUserHandler(userSvc)

	// Gin 라우터 생성 및 라우팅 구성
	r := gin.Default()
	router.Register(r, router.Dependencies{
		UserHandler: userH,
	})

	addr := ":" + cfg.Port
	log.Printf("starting server at %s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
