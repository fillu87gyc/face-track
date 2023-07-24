package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/fillu87gyc/face-track/config"
	logger "github.com/fillu87gyc/lambda-go/lib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Pose struct {
	Pose    string  `json:"pose"`
	DoTime  float64 `json:"do_time"`
	NodFlag bool    `json:"nod_flag"`
}

func main() {
	go nodRoutine()
	go trackRoutine()
	logger := logger.GetLogger()
	e := gin.New()
	e.Use(MiddleWareLogger(logger))
	e.Use(gin.Recovery())
	e.POST("/drive/", drive)
	e.GET("/facepos/:x/:y", okao)
	_ = e.Run(":3333")
}

var faceX float64 = 0.5
var faceY float64 = 0.5

func okao(c *gin.Context) {
	x := c.Param("x")
	y := c.Param("y")
	faceX, _ = strconv.ParseFloat(x, 64)
	faceX = 1 - faceX
	faceY, _ = strconv.ParseFloat(y, 64)
	// logger.GetLogger().Info("今顔の座標を受け取った" + x + " " + y)
	c.JSON(http.StatusOK, gin.H{"x": x, "y": y})
}

var nodFlag bool = false
var trackFlag bool = false

func nodRoutine() {
	// 3秒ごとにHTTP GETを送信
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	// 無限ループで定期的に処理を行う
	for range ticker.C {
		if nodFlag {
			fmt.Println("Nod!")
			_ = SendPose("nod")
		}
	}
}

func SendPose(pose string) error {
	url := config.MotorServerURL
	query := fmt.Sprintf("/takubo/preset/%s", pose)
	resp, err := http.Get(url + query)

	if pose != "nod" {
		logger.GetLogger().Info("今poseおくった")
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK || string(body) != "OK" {
		return fmt.Errorf("Error: %v, StatusCode: %v, Body: %v", err, resp.StatusCode, string(body))
	}
	return nil
}

func trackRoutine() {
	ticker := time.NewTicker(50 * time.Microsecond)
	defer ticker.Stop()
	// 無限ループで定期的に処理を行う
	for range ticker.C {
		if trackFlag {
			//顔の座標情報を取得
			url := config.MotorServerURL
			query := fmt.Sprintf("/takubo/pose/%v/%v/%v", faceX, faceY, -1)
			if _, err := http.Get(url + query); err != nil {
				fmt.Println("Error: ", err)
			}
			if trackFlag {
				logger.GetLogger().Info("今顔おくった :" + query)
			} else {
				logger.GetLogger().Info("みつけてよかったばぐのもと")
			}
		}
	}
}

func drive(c *gin.Context) {
	var data []Pose
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "JSON data processed successfully"})
	c.Abort()

	go func() {
		// データの処理
		for _, pose := range data {
			fmt.Printf("Pose: %s, DoTime: %v, NodFlag: %v\n", pose.Pose, pose.DoTime, pose.NodFlag)
			nodFlag = pose.NodFlag
			trackFlag = (pose.Pose == "track")
			logger.GetLogger().Info("今待機")
			time.Sleep(500 * time.Microsecond) //状態が切り替わるまで待つ
			logger.GetLogger().Info("今待機おわり")
			if trackFlag {
				time.Sleep(time.Duration(pose.DoTime) * time.Second)
			} else {
				// presetの呼び出し
				// あとでinjection仕様にする
				ch := make(chan struct{})
				go func(ch <-chan struct{}) {
					for {
						select {
						case <-ch:
							println("owari")
							return
						default:
							if err := SendPose(pose.Pose); err != nil {
								fmt.Println("Error: ", err)
							}
							time.Sleep(100 * time.Millisecond)
							println("aa")
						}
					}
				}(ch)

				time.Sleep(time.Duration(pose.DoTime) * time.Second)
				ch <- struct{}{}
			}
		}
	}()
}

func MiddleWareLogger(l *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// l.Info("",
		// 	zap.Int("status", c.Writer.Status()),
		// 	zap.String("method", c.Request.Method),
		// 	zap.String("path", c.Request.URL.Path),
		// 	zap.String("query", c.Request.URL.RawQuery),
		// 	zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		// )
	}
}
