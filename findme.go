package main

import (
	"bufio"
	json2 "encoding/json"
	"fmt"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/bitly/go-simplejson"
	"github.com/kbinani/screenshot"
	"golang.design/x/hotkey"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var a = app.New()
var w = a.NewWindow("heybcat/findme")
var entry = widget.NewEntry()

var show = false
var curDir = ""

func doHide() {
	w.Hide()
	show = false
	entry.SetText("")
}

func doShow() {
	w.Show()
	show = true
}

func save(img *image.RGBA, filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	png.Encode(file, img)
}

func doFind(text string) {
	// screenshot first
	displays := screenshot.NumActiveDisplays()
	if displays < 0 {
		return
	}
	for i := 0; i < displays; i++ {
		img, err := screenshot.CaptureDisplay(i)

		if err != nil {
			return
		}
		save(img, strconv.Itoa(i)+"screenshot.png")
		doOcr(strconv.Itoa(i)+"screenshot.png", text)
	}
}

func doOcr(fileName string, targetText string) {
	cmd := exec.Command(curDir+"\\ocr\\RapidOCR-json.exe", "--models", curDir+"\\ocr\\models", "--image", curDir+"\\"+fileName, "--unClipRatio", "1.2")

	// 获取子进程的标准输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error getting StdoutPipe:", err)
		return
	}

	// 启动子进程
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting command:", err)
		return
	}

	// 使用 bufio.Scanner 读取子进程的标准输出
	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			out := scanner.Text()
			fmt.Println(out)
			if strings.HasPrefix(out, "{") {
				fmt.Println("have result")
				json, jsonErr := simplejson.NewJson([]byte(out))
				if jsonErr != nil {
					continue
				}
				code, _ := json.Get("code").Int()
				if code != 100 {
					continue
				}
				data, _ := json.Get("data").Array()
				file, fileErr := os.Open(fileName)
				if fileErr != nil {
					continue
				}
				defer file.Close()

				img, err := png.Decode(file)
				if err != nil {
					continue
				}
				bounds := img.Bounds()
				drawImg := image.NewRGBA(bounds)
				draw.Draw(drawImg, bounds, img, bounds.Min, draw.Src)

				// {"box":[[2037,5],[2170,5],[2170,48],[2037,48]],"score":0.22402291372418404,"text":"Ga"}
				for _, row := range data {
					if eachMap, ok := row.(map[string]interface{}); ok {
						if score, ok := eachMap["score"].(float64); ok {
							if score < 0.8 {
								continue
							}
						}
						if text, ok := eachMap["text"].(string); ok {
							if strings.Contains(text, targetText) {
								box := eachMap["box"].([]interface{})
								if len(box) == 4 {
									x1, _ := box[0].([]interface{})[0].(json2.Number).Int64()
									//y1, _ := box[1].([]interface{})[1].(json2.Number).Int64()
									x2, _ := box[2].([]interface{})[0].(json2.Number).Int64()
									y2, _ := box[3].([]interface{})[1].(json2.Number).Int64()
									x1_, _ := strconv.Atoi(strconv.FormatInt(x1, 10))
									//y1_, _ := strconv.Atoi(strconv.FormatInt(y1, 10))
									x2_, _ := strconv.Atoi(strconv.FormatInt(x2, 10))
									y2_, _ := strconv.Atoi(strconv.FormatInt(y2, 10))
									drawLine(drawImg, x1_, y2_, x2_, y2_, color.RGBA{255, 0, 0, 255})
								}
								fmt.Println(box)
								fmt.Println("find text:", text)
							}
						}
					}
				}
				// 保存绘制后的图片

				outFile, err := os.Create(fileName + "-output.jpg")

				if err != nil {

					panic(err)

				}

				defer outFile.Close()

				jpeg.Encode(outFile, drawImg, &jpeg.Options{Quality: 100})
				openImg(fileName + "-output.jpg")
			}

		}
	}()

	// 等待子进程完成
	if err := cmd.Wait(); err != nil {
		fmt.Println("Error waiting for command:", err)
	}
}

// 在图片上绘制矩形

func drawRectangle(img *image.RGBA, x1, y1, x2, y2 int, color color.RGBA) {
	draw.Draw(img, image.Rect(x1, y1, x2, y2), &image.Uniform{color}, image.ZP, draw.Src)
}

// 辅助函数：获取值的绝对值

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// 在图片上绘制直线
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, color color.RGBA) {

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 0, 0
	if x0 < x1 {
		sx = 1
	} else {
		sx = -1
	}
	if y0 < y1 {
		sy = 1
	} else {
		sy = -1
	}
	err := dx - dy
	for {
		img.Set(x0, y0, color)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}

}

func openImg(img string) {
	cmd := exec.Command("cmd", "/C", "start", "/B", curDir+"\\"+img)
	err := cmd.Start()
	if err != nil {
		return
	}

}

func main() {

	dir, err := os.Getwd()
	if err != nil {
		fmt.Print("error:", err)
		return
	}

	//dir = strings.Replace(dir, "\\", "/", -1)
	fmt.Println("current dir:", dir)
	curDir = dir

	label := widget.NewLabel("Input what to find!")
	button := widget.NewButton("find", func() { label.SetText("Find :)") })
	w.SetContent(container.NewVBox(label, entry, button))
	a.Lifecycle().SetOnExitedForeground(func() { doHide() })

	go func() {
		// Register a desired hotkey.
		hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyS)
		if err := hk.Register(); err != nil {
			panic("hotkey registration failed")
		}
		// Start listen hotkey event whenever it is ready.
		for range hk.Keydown() {
			if show {
				doHide()
			} else {
				doShow()
			}
		}
	}()

	button.OnTapped = func() {
		text := entry.Text
		doHide()
		time.Sleep(500 * time.Millisecond)
		doFind(text)
	}

	//w.SetCloseIntercept(func() { doHide() })
	w.SetFixedSize(true)
	w.CenterOnScreen()
	a.Run()

}
