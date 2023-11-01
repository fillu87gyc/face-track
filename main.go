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
	go toDefaultPotision()
	logger := logger.GetLogger()
	e := gin.New()
	e.Use(MiddleWareLogger(logger))
	e.Use(gin.Recovery())
	e.POST("/drive/", drive)
	e.GET("/facepos/:x/:y", okao)
	_ = e.Run(":3333")
}

var faceX int = maxX / 2
var faceY int = maxY / 2

var ix = maxX / 2
var iy = maxY / 2

const (
	//どれくらい中央として判定するかの許容値
	allowRangeX = 30
	allowRangeY = 10
	fineRange   = 30
	xCourseGain = 10 //ざっくり
	yCourseGain = 15 //ざっくり
	xFineGain   = 6  //細かく
	yFineGain   = 10 //細かく
	maxX        = 640
	maxY        = 480
)

func isRangeX(x, maxX int) bool {
	return x < maxX/2+allowRangeX && x > maxX/2-allowRangeX
}
func isRangeY(y, maxY int) bool {
	return y < maxY/2+allowRangeY && y > maxY/2-allowRangeY
}

func toDefaultPotision() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	oldX, oldY := maxX/2, maxY/2
	for range ticker.C {
		if faceX == oldX && faceY == oldY && !nodFlag {
			//一定期間止まってたのででdefaultに戻す
			faceX = maxX / 2
			faceY = maxY / 2
		} else {
			oldX = faceX
			oldY = faceY
		}
	}
}

func okao(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"x": faceX, "y": faceY})
	c.Abort()

	if !trackFlag {
		logger.GetLogger().Info("追従じゃないので無視")
		return
	}
	x := c.Param("x")
	y := c.Param("y")
	ix, _ = strconv.Atoi(x)
	iy, _ = strconv.Atoi(y)
	updateFaceXY()
}

// 近くにいるかどうか
func isInNearRange(pos, MAX int) bool {
	r := fineRange + allowRangeX
	inRange := pos < MAX/2+r && pos > MAX/2-r
	logger.GetLogger().Info(fmt.Sprintf("%d is near? =  %v", pos, inRange))
	return inRange
}
func updateFaceXY() {
	if isRangeX(ix, maxX) && isRangeY(iy, maxY) {
		logger.GetLogger().Info("jast mannaka!!")
	} else {
		if isRangeY(ix, maxX) {
			//顔のx座標が中央にある
			logger.GetLogger().Info("顔のx座標が中央にある")
		} else {
			//顔のx座標が中央にない
			if ix < maxX/2 {
				//顔が右にある
				if isInNearRange(ix, maxX) {
					faceX += xFineGain
				} else {
					faceX += xCourseGain
				}
				if faceX > maxX {
					faceX = maxX
				}
			} else {
				//顔が左にある
				if isInNearRange(ix, maxX) {
					faceX -= xFineGain
				} else {
					faceX -= xCourseGain
				}
				if faceX < 0 {
					faceX = 0
				}
			}
		}

		if isRangeY(iy, maxY) {
			//顔のy座標が中央にある
			logger.GetLogger().Info("顔のy座標が中央にある")
		} else {
			//顔のy座標が中央にない
			if iy < maxY/2 {
				//顔が下にある
				if isInNearRange(iy, maxY) {
					faceY += yFineGain
				} else {
					faceY += yCourseGain
				}
				if faceY > maxY {
					faceY = maxY
				}
			} else {
				//顔が上にある
				if isInNearRange(iy, maxY) {
					faceY -= yFineGain
				} else {
					faceY -= yCourseGain
				}
				if faceY < 0 {
					faceY = 0
				}
			}
		}
	}
}

var nodFlag bool = false
var trackFlag bool = true

func nodRoutine() {
	// 3秒ごとにHTTP GETを送信
	ticker := time.NewTicker(2 * time.Second)
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
	if pose != "nod" {
		logger.GetLogger().Info("今poseおくった")
	}
	resp, err := http.Get(url + query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK || string(body) != "OK" {
		return fmt.Errorf("error: %s, statusCode: %d, body: %s", err, resp.StatusCode, string(body))
	}
	return nil
}

func trackRoutine() {
	// ticker := time.NewTicker(20 * time.Microsecond)
	// defer ticker.Stop()
	// 無限ループで定期的に処理を行う
	faceX = maxX / 2
	faceY = maxY / 2
	for {
		if trackFlag {
			//顔の座標情報を取得
			go func() {
				url := config.MotorServerURL
				query := fmt.Sprintf("/takubo/pose/%v/%v/%v", float64(faceX)/maxX, float64(faceY)/maxY, 0.5)
				resp, err := http.Get(url + query)
				if err != nil {
					fmt.Println("Error: ", err)
					return
				}
				defer resp.Body.Close()
				logger.GetLogger().Info("trackRoutine: " + query)
			}()
		}
		time.Sleep(100 * time.Millisecond)
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
