# face track

これはtakuboでコマンドと追従式のカメラを使うことで任意のコマンドを実行できる様になるもの

```go
e.POST("/drive/", drive)
e.GET("/facepos/:x/:y", okao)
```

この後ろに実際にdynamixelを制御するノードを置く
