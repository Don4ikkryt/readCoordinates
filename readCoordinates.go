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
	Northest       Point
	Southest       Point
	Eastest        Point
	Westest        Point
)

const (
	lenghtOfEquatorInMeters float64 = 40000000.
	circleDegrees           float64 = 360.
	minutesInDegree                 = 21600.
	secondsInDegree                 = 1296000.
	jpgRegexp                       = `.*\.(jpg|JPG)$`
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
	findNSWE(points)
	LatitudeDif := coordinateDiffernce(Northest., Southest)
	LongtitudeDif := coordinateDiffernce(Westest, Eastest)
	fmt.Println(LatitudeDif)
	fmt.Println(LongtitudeDif)

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
func findNSWE(points []Point) {
	i := 1
	for _, value := range points {

		if i == 1 {
			Northest = value
			Westest = value
			Southest = value
			Eastest = value
		} else {

			if ifLatitude1BiggerLatitude2(value, Northest) {
				Northest = value
			}

			if ifLatitude1BiggerLatitude2(Southest, value) {
				Southest = value
			}

			if ifLongtitude1BiggerLongtitude2(value, Eastest) {
				Eastest = value
			}

			if ifLongtitude1BiggerLongtitude2(Westest, value) {
				Westest = value

			}

		}
		i++
	}
}

func ifLatitude1BiggerLatitude2(point1 Point, point2 Point) bool {
	i := 0
	if len(point1.Latitude) != len(point1.Latitude) {
		fmt.Println("Different length of coordinate")
		return false
	}
	for _, value := range point1.Latitude {
		if value != point2.Latitude[i] {

			if value > point2.Latitude[i] {

				return true
			} else {

				return false
			}
		}
		i++

	}
	return false
}
func ifLongtitude1BiggerLongtitude2(point1 Point, point2 Point) bool {
	i := 0
	if len(point1.Longtitude) != len(point1.Longtitude) {
		fmt.Println("Different length of coordinate")
		return false
	}
	for _, value := range point1.Longtitude {
		if value != point2.Longtitude[i] {

			if value > point2.Longtitude[i] {

				return true
			} else {

				return false
			}

		}
		i++

	}
	return false
}

func coordinateDiffernce(coordinates1 []float64, coordinates2 []float64) (difference []float64) {
	if len(coordinates1) != len(coordinates2) {
		fmt.Println("Different length of coordinate")
		return nil
	}
	bigger := coordinates1
	smaller := coordinates2
	for i := 0; i < len(coordinates1); i-- {
		if coordinates1[i] != coordinates2[i] {
			if coordinates1[i] > coordinates2[i] {
				bigger = coordinates1
				smaller = coordinates2
			} else {
				bigger = coordinates2
				smaller = coordinates1
			}
		}
	}
	var result [3]float64
	for i := len(coordinates1) - 1; i >= 0; i-- {
		result[i] = bigger[i] - smaller[i]
		if result[i] < 0 {
			result[i] = 60 + bigger[i] - smaller[i]
			if bigger[i+1] == 0 {
				bigger[i+1] = 59
				bigger[i+2]--
			} else {
				bigger[i+1]--
			}

		}
	}
	difference = append(difference, result[0], result[1], result[2])
	return
}
func convertFromCoordinatesToMeterLatitude(coordinates []float64) (meters float64) {
	meters = coordinates[0]*lenghtOfEquatorInMeters/circleDegrees + coordinates[1]*lenghtOfEquatorInMeters/minutesInDegree + coordinates[2]*lenghtOfEquatorInMeters/secondsInDegree
	return
}
