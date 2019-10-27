package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
)

var (
	sourceFolder   string
	filteredFolder string
)

const (
	jpgRegexp = `.*\.(jpg|JPG)$`
)

type Point struct {
	Longtitude []float64
	Latitude   []float64
	Filename   string
}

func newPoint(longtitude [3]float64, latitude [3]float64, filename string) *Point {

	p := Point{Filename: filename}
	p.Longtitude = append(p.Longtitude, longtitude[0], longtitude[1], longtitude[2])
	p.Latitude = append(p.Latitude, latitude[0], latitude[1], latitude[2])
	return &p
}

func main() {
	parseFlags()
	createFilteredFolder(filteredFolder)

	if sourceFolder == "" ||
		filteredFolder == "" {
		log.Fatal("ERROR: all flags must be set")
	}

	exif.RegisterParsers(mknote.All...)

	filenames := getFilenames(sourceFolder)
	filesToPosition := mapFilesToCoordinates(filenames)
	points := createPointArray(filenames, filesToPosition)
	for _, value := range points {
		fmt.Println(value)
	}
	log.Print("Successfully filtered photos!")
}

func parseFlags() {
	flag.StringVar(&sourceFolder, "source_folder", "", "Path to the folder with photos")
	flag.StringVar(&filteredFolder, "filtered_folder", "", "Path to the folder with filtered (unsupported format/no exif data) files")

	flag.Parse()
}

func createFilteredFolder(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}
}

func getFilenames(path string) []string {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	filenames := make([]string, 0)
	re := regexp.MustCompile(jpgRegexp)
	for _, file := range files {
		if re.MatchString(file.Name()) {
			filenames = append(filenames, file.Name())
		}
	}

	return filenames
}

func mapFilesToCoordinates(filenames []string) map[string][2][3]float64 {
	position := make(map[string][2][3]float64)
	for _, filename := range filenames {
		LatitudeAndLongtitude, err := retrievePosition(sourceFolder, filename)
		if err != nil {
			moveToFilteredFolder(sourceFolder, filename)
			continue
		}
		position[filename] = LatitudeAndLongtitude

	}

	return position
}

func decodeImg(file *os.File, path string, filename string) (*exif.Exif, error) {
	exifFile, err := exif.Decode(file)
	if err != nil {
		moveToFilteredFolder(path, filename)
		return nil, err
	}

	return exifFile, nil
}

func getGPSLatitudeAndGPSLongtitude(file *os.File, path, filename string) (tiffLongitude *tiff.Tag, tiffLatitude *tiff.Tag, err error) {
	exifFile, err := decodeImg(file, path, filename)
	if err != nil {
		return nil, nil, err
	}

	tiffLongitude, err = exifFile.Get(exif.GPSLongitude)
	if err != nil {
		moveToFilteredFolder(path, filename)
		return nil, nil, err
	}
	tiffLatitude, err = exifFile.Get(exif.GPSLatitude)

	if err != nil {
		moveToFilteredFolder(path, filename)
		return nil, nil, err
	}

	return
}

func retrievePosition(path, filename string) (position [2][3]float64, err error) {
	image, err := os.Open(path + "\\" + filename)

	if err != nil {
		return
	}
	defer image.Close()

	tiffLongitude, tiffLatitude, err := getGPSLatitudeAndGPSLongtitude(image, path, filename)
	if err != nil {
		return
	}
	position[0] = fromStringToFloat(tiffLongitude.String())
	position[1] = fromStringToFloat(tiffLatitude.String())
	return
}

func moveToFilteredFolder(path string, filename string) {
	oldPlace := path + "\\" + filename
	newPlace := filteredFolder + "\\" + filename
	if err := os.Rename(oldPlace, newPlace); err != nil {
		log.Fatalf("failed to move file to filtered folder: %s", err.Error())
	}
}

func fromStringToFloat(position string) (returning [3]float64) {
	temp := strings.Trim(position, "\"[]")
	substrings := strings.Split(temp, "\",\"")

	i := 0
	for _, value := range substrings {
		temp := strings.Split(value, "/")
		divident, err := strconv.ParseFloat(temp[0], 64)
		if err != nil {
			log.Fatal(err)
		}
		divider, err := strconv.ParseFloat(temp[1], 64)
		if err != nil {
			log.Fatal(err)
		}
		returning[i] = divident / divider
		i++
	}
	return
}

func createPointArray(filenames []string, position map[string][2][3]float64) (points []Point) {
	for _, value := range filenames {
		NewPoint := newPoint(position[value][0], position[value][1], value)
		points = append(points, *NewPoint)
	}
	return

}
